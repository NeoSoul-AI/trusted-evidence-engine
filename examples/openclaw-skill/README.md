# OpenClaw Skill Example

This folder is the placeholder for the OpenClaw integration layer.

The OpenClaw packaging model should remain:

- `SKILL.md`
- `scripts/`
- `assets/`
- `references/`

The skill should call Trusted Evidence Engine over CLI or HTTP and pass the returned `evidence pack` into OpenClaw strategy logic.

The core Go engine should not be embedded directly into the skill bundle.
