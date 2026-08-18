# Deployment Kit Third-Party Sources

The RustFS secret-file and container-volume conventions in this deployment kit were verified against the official RustFS documentation on 2026-08-18.

| Source | URL | Applied boundary |
| --- | --- | --- |
| RustFS Docker installation | https://docs.rustfs.com/en/installation/container | Single-node container command shape, persistent `/data` volume, loopback port exposure, and non-default credentials. |
| RustFS environment variables | https://docs.rustfs.com/en/reference/environment-variables | `RUSTFS_ACCESS_KEY_FILE`, `RUSTFS_SECRET_KEY_FILE`, listener addresses, and health endpoint conventions. |
