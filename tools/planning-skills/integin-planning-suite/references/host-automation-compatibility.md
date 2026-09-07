# Host Automation Compatibility Contract

This package does not enable lifecycle hooks, automatic context injection, or forced completion gates. A future host integration must provide all of the following before those features can be considered:

1. An official, versioned hook schema and event contract.
2. An explicit project-root binding that cannot fall back to an unrelated directory.
3. An opt-in and opt-out path visible to the owner.
4. A bounded content policy, redaction rules, and a record of what was injected.
5. A failure mode that cannot block a user indefinitely.
6. A testable override and recovery path.

Until that contract is verified and separately approved, use `render-context` and `check-complete` as manual equivalents.
