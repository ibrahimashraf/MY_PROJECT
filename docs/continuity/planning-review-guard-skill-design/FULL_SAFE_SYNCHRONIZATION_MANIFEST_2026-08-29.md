# Full Safe Synchronization Manifest

## Authorization

The owner approved synchronization of all locally created, non-secret materials to `C:\MY_PROJECT`. This manifest defines a non-destructive copy into dedicated locations. Existing project files must not be overwritten.

## Included source families

| Source | Destination under `C:\MY_PROJECT` | Copy rule |
|---|---|---|
| `/home/ubuntu/review-guard-skill-design/*.md` and non-secret checksum/manifests | `docs\continuity\planning-review-guard-skill-design\` | Preserve file names; do not overwrite existing files. |
| `/home/ubuntu/skills/integin-planning-suite/` | `tools\planning-skills\integin-planning-suite\` | Preserve package structure; locally owned clean-room package. |
| `/home/ubuntu/skills/integin-planning-orchestrator/` | `tools\planning-skills\integin-planning-orchestrator\` | Preserve package structure; rollback/source package copy. |
| `/home/ubuntu/skills/planning-with-files/` | `tools\planning-skills\planning-with-files\` | Preserve package structure; predecessor rollback copy. |
| Existing PC bridge and launcher records | Existing `tools\planning-bridge\`, `tools\planning-task-launcher\`, and operations documentation | Verify only; do not overwrite or re-enable. |

## Explicit exclusions

No files or directories under private material, credentials, environment files, secrets, active runtime state, databases, sockets, logs, browser profiles, external source repositories, external prompts/hooks/scripts, build caches, `node_modules`, or generated `__pycache__` directories may be copied. No file may be copied into product source, protected operations controls, or runtime configuration merely because it is locally available.

## Safety rules

The copy must use an existing-file-preserving operation. It must not delete, overwrite, rename, merge, execute, install, schedule, start, bind, or enable anything. After copying, checksums and file counts must be recorded, and protected paths must be checked for unchanged state. The bridge and launcher remain disabled.

## Verification requirements

The final manifest must list copied files and SHA-256 hashes, identify skipped existing files, confirm excluded patterns were not copied, and confirm no runtime or private-material path was accessed.
