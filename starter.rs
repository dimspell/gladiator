use tokio::process::Command;

#[tokio::main]
async fn main() {
  let mut child = Command::new("my_program")
      .spawn()
      .expect("failed to execute process");

  let _ = child.wait().await;
  
  // continue doing other work  
}