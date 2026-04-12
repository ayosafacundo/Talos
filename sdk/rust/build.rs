fn main() -> Result<(), Box<dyn std::error::Error>> {
    let proto = "../../api/proto/talos/hub/v1/hub.proto";
    let include = "../../api/proto";
    println!("cargo:rerun-if-changed={proto}");
    tonic_build::configure().build_server(false).compile_protos(&[proto], &[include])?;
    Ok(())
}
