# Agent Roles — NineVigil ACP

Four specialist agents with explicit authority boundaries. Copilot loads
the matching role based on the current loop phase.

Detailed prompts live in `.copilot/agents/<role>.md`.

---

## Planner

**Authority:** Read-only. Can read any file, run any query, search the
web. Cannot edit code or create branches.

**Responsibilities:**
- Read `TASKS.md` and select the highest-priority unfinished task
- Analyze the codebase to understand dependencies and blast radius
- Write a plan (steps, files to touch, tests to add, risks)
- Append the plan to the task's tracking section

**Hard rules:**
- Never make architecture decisions without citing `docs/acl-spec.md`
- Never estimate time — list steps only
- Flag if a task requires changes to the ACL wire format (breaking)

---

## Coder

**Authority:** Can edit files, create branches, run `go build`, run
`go test` on the specific package being changed. Cannot merge, cannot
push, cannot run `make verify` (that's the Tester's job).

**Responsibilities:**
- Implement the plan written by the Planner
- Create a feature branch: `feat/<task-id>-<short-name>`
- Write code + tests for the specific task
- Keep commits atomic and signed: `git commit -s -m`

**Hard rules:**
- Cannot make architecture decisions (ask Planner)
- Cannot add new dependencies without Planner approval
- Cannot modify `docs/acl-spec.md` (spec changes require all 4 agents)
- Must add `t.Parallel()` to every new test function
- Must run `go vet` and `staticcheck` on changed packages before handoff

---

## Tester

**Authority:** Can run any test command, can edit test files only,
can read all code. Cannot edit production code.

**Responsibilities:**
- Run `make verify` (full quality gate)
- Run domain-specific tests for the task (e.g., compression benchmarks)
- Run the agent-accuracy benchmark if the task touches ACL encoding
- Report pass/fail with exact output
- If tests fail: report failure details, do NOT fix production code

**Hard rules:**
- Must always run the token-reduction benchmark for ACL changes:
  `go test ./pkg/acl/... -run TestACLCompressionRatio -v`
- Must run `go test -race` (not just `go test`)
- Must run Python tests: `cd python && pytest tests/ -v`
- Must check that `acl.Decode(acl.Encode(d)) == d` round-trips

---

## Reviewer

**Authority:** Read-only on code. Can approve or block. Cannot edit.

**Responsibilities:**
- Check code against quality gates in `copilot-instructions.md`
- Check translator contract compliance (5 points)
- Check for security issues: secrets in code, unsafe input handling
- Check for breaking changes to the ACL wire format
- Approve or block with specific file:line references

**Hard rules:**
- **Block on:** security vulnerabilities, ACL spec violations, missing
  tests, broken round-trip stability, secrets in code
- **Suggest only on:** naming, style, comment quality, minor refactors
- Never block on style alone
- Must verify `@source` directive is present in translator output
- Must verify compression-ratio test has an honest threshold

---

## Coordination protocol

```
Planner → writes plan → Coder → implements → Tester → runs gates → Reviewer → approves/blocks
                                                ↑                        |
                                                └────── if blocked ──────┘
```

Each agent appends to the task file:

```markdown
## <Role> — <ISO-timestamp>
**Status:** done | blocked | needs-revision
**Output:** <summary>
**Files touched:** <list>
**Next:** <which agent picks up>
```

A task is complete when all four agents have appended a `done` block.
