<!-- BEGIN RELYNCE MANAGED BLOCK - DO NOT EDIT -->
## Relynce

You have access to the Relynce reliability platform via the `rely` CLI and expert agents.

### Context Tools (always available)

Use these CLI commands when reliability context would inform the conversation:

**Risk Data:**
- `rely risk list --service={service} --limit=50` — Current risks for a service
- `rely risk show R-XXX` — Full risk details with mapped controls
- `rely risk context R-XXX --format=json` — Risk + controls + knowledge + incidents
- `rely risk ready --service={service}` — Risks ready to remediate (no blockers)

**Control Catalog:**
- `rely control list --limit=100` — All reliability controls
- `rely control show RC-XXX` — Specific control details and evidence status

**Knowledge Base:**
- `rely knowledge search "{query}" --limit=5` — Search incidents and patterns
- `rely knowledge enrich --query="{query}"` — Enriched context with patterns and procedures

**Evidence & Resolution:**
- `rely evidence submit --control=RC-XXX --type=code --name="..." --url="..." --description="..."` — Record implementation evidence
- `rely risk resolve R-XXX --reason="..."` — Mark a risk as resolved

### Expert Routing (ambient invocation)

When the conversation enters a reliability domain and the user is making design decisions or implementing non-trivial patterns, you can invoke a domain expert agent via the Task tool. Use your judgment — not every mention of a topic needs an expert.

| Domain | Agent | When to Invoke |
|--------|-------|----------------|
| Observability, tracing, metrics, logging | `observability-pro` | Designing instrumentation or alerting strategy |
| Circuit breakers, retries, timeouts, fallbacks | `resilience-pro` | Implementing fault tolerance patterns |
| CI/CD pipelines, deployment gates, automation | `cicd-pro` | Designing deployment pipeline or build process |
| LLM/AI integration, prompts, eval pipelines | `ai-reliability-pro` | Building AI features or LLM integrations |
| Capacity planning, autoscaling, query optimization | `capacity-planning-pro` | Sizing resources, optimizing queries, load testing |
| Cloud costs, budgets, quotas | `cost-governance-pro` | Cost optimization or budget planning |
| Blue-green, canary, GitOps, rollback | `deployment-excellence-pro` | Designing rollout or rollback strategy |
| Test pyramid, coverage, chaos testing | `development-testing-pro` | Designing test strategy or improving coverage |
| DB backup, PITR, disaster recovery | `disaster-recovery-pro` | Backup strategy or DR planning |
| On-call, runbooks, escalation | `incident-response-pro` | Setting up on-call or incident procedures |
| Postmortems, blameless culture, learning | `post-incident-pro` | Running postmortems or improving incident learning |
| Risk register, error budgets, reliability culture | `reliability-culture-pro` | Reliability planning or organizational practices |
| Dependency scanning, SBOM, supply chain, auth | `security-supply-chain-pro` | Security review or supply chain hardening |
| SLI/SLO definition, burn rate alerts | `slo-monitoring-pro` | Defining SLOs or setting up error budget alerts |
| Go reliability patterns | `golang-pro` | Go-specific reliability implementation |
| JavaScript/TypeScript reliability | `javascript-pro` | JS/TS-specific reliability patterns |
| Python reliability patterns | `python-pro` | Python-specific reliability implementation |

**When NOT to invoke ambiently:**
- Simple factual questions the AI can answer from general knowledge
- The user is clearly focused on non-reliability work
- The topic was already covered by a recent `/rely:ask` invocation

### Skill Reference

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `/rely:scan` | Multi-agent codebase risk scan | Before pushing code, after significant changes, periodic assessment |
| `/rely:fix R-XXX` | Guided risk remediation with expert consultation | When you have a specific risk to address |
| `/rely:ask <question>` | Expert consultation with auto-routing | Any reliability question — "how do I implement X?" |
| `/rely:risks` | Risk register view (posture, ready, full) | Starting a session, checking what needs attention |
| `/rely:review` | Reliability-focused code review | Before merge, reviewing current changes |
| `/rely:evidence` | Submit implementation evidence | After implementing a control or fixing a risk |
| `/rely:status` | CLI and API health check | Troubleshooting connectivity |
<!-- END RELYNCE MANAGED BLOCK -->
