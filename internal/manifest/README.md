# Manifest Builder

Responsible for building token-minimal Execution Manifests from registry tools,
resolved capabilities, policy, and constraints.

Week 1 scope:

- Accept selected tools
- Strip schemas to compact field/type maps
- Emit deterministic action ids
- Compute simple `depends_on` ordering
