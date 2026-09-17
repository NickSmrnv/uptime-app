# Repository Guidelines

## Workspace Structure

This workspace contains two independent applications:

- `frontend/` contains the web client. See `frontend/AGENTS.md` before changing it.
- `backend/` contains the API. See `backend/AGENTS.md` before changing it.

Run commands from the relevant application directory. Do not mix JavaScript dependencies, generated files, or Go modules between the two projects.

## When starting a new task use GitHub Flow

Details in files docs/rules/github-flow.md

## When creating Pull Requests

Use rules in files docs/rules/pr.md

## When commiting

Use rules in files docs/rules/commit.md

## Validation

Before review, run the checks specified by the local `AGENTS.md` for every application changed. Avoid unrelated formatting or dependency updates.

## Комментарии

- Комментируй "почему", а не "что", так как это видно из кода
- Очевидное не комментируй, лучше используй правильные наименования функций
- Публичные функции doc comment
- Сложную арифметику поясняй рядом
- Меняешь код - актуализируй комментарий