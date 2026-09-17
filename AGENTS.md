# Repository Guidelines

## Workspace Structure

This workspace contains two independent applications:

- `frontend/` contains the web client. See `frontend/AGENTS.md` before changing it.
- `backend/` contains the API. See `backend/AGENTS.md` before changing it.

Run commands from the relevant application directory. Do not mix JavaScript dependencies, generated files, or Go modules between the two projects.

## Shared Contribution Rules

Keep commits focused and use short imperative subjects, for example `Add uptime check handler`. The available workspace snapshot has no accessible Git history, so no project-specific convention can be verified.

Pull requests should explain the user-facing change, list validation performed, link the relevant issue, and call out configuration changes or deferred tests. Include screenshots for visual frontend changes.

## Validation

Before review, run the checks specified by the local `AGENTS.md` for every application changed. Avoid unrelated formatting or dependency updates.

## Коммиты

- Формат Conventional Commits (feat, fix, refactor, test, docs, chore);
- Заголовок до 72 символов, в повелительном наклонении;
- Без эмодзи, без "significantly improved" и прочей воды;
- Тело – только если нужно объяснять "почему", а не "что";
- Один логический шаг – один коммит.