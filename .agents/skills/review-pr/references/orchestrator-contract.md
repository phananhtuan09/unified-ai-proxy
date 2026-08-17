# Orchestrator Contract

Apply this contract only when an orchestrator explicitly invokes `review-pr` and provides a feature slug or review output path.

Use the supplied output path.

If no path is supplied but a feature slug is available, write `docs/ai/reviews/{feature}.md`.

Append exactly one HTML comment as the final output line:

- `Ready for Human PR Approval`: `<!-- orchestrator: outcome=continue provides=pr_review_path pr_review_path={review-path} -->`
- `Needs Fix`: `<!-- orchestrator: outcome=stop-fail -->`
- `Needs Human Decision`: `<!-- orchestrator: outcome=stop-ask-human -->`
- `Blocked`: `<!-- orchestrator: outcome=stop-blocked -->`
