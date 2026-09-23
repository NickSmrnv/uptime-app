# Repository Guidelines

## Workspace Structure

This workspace contains two independent applications:

- `frontend/` contains the web client. See `frontend/AGENTS.md` before changing it.
- `backend/` contains the API. See `backend/AGENTS.md` before changing it.

Run commands from the relevant application directory. Do not mix JavaScript dependencies, generated files, or Go modules between the two projects.

## Shared Contribution Rules

Keep commits focused and use short imperative subjects, for example `Add uptime check handler`. The available workspace snapshot has no accessible Git history, so no project-specific convention can be verified.

Pull requests should explain the user-facing change, list validation performed, link the relevant issue, and call out configuration changes or deferred tests. Include screenshots for visual frontend changes.

## GitHub Flow

- Перед началом каждой задачи проверить текущую ветку и рабочее дерево. Если работа всё ещё находится в ветке предыдущей задачи, не начинать новую задачу и не создавать новую ветку: сначала показать пользователю оставшиеся изменения и запросить подтверждение на их завершение и коммиты.
- После подтверждения завершить проверки и создать логические коммиты предыдущей задачи. Затем отдельно запросить подтверждение на слияние и публикацию этой работы в `master`. Новую ветку создавать только после успешного обновления `master` и перехода на чистую актуальную `master`.
- Перед началом каждой задачи создать отдельную ветку от актуальной `master`; не вносить изменения и не создавать коммиты напрямую в `master`, кроме явно подтверждённого пользователем завершения предыдущей задачи или работы со skills.
- Имена веток: `feat/<feature-name>` для новой функциональности и `fix/<fix-name>` для исправлений.
- Название после префикса писать на английском, в `kebab-case`, например: `feat/avatar-upload` или `fix/avatar-display`.
- После выполнения задачи открыть Pull Request из рабочей ветки в `master`; вливать изменения только через Pull Request после прохождения обязательных проверок, если пользователь явно не подтвердил прямое завершение предыдущей задачи в `master`.

## Pull Requests

Перед созданием Pull Request:

- Убедиться, что ветка создана от актуального `master`, а все изменения задачи находятся только в ней. Не включать в PR пользовательские или несвязанные изменения без явного запроса.
- Проверить `git diff --check`, просмотреть `git diff master...HEAD` и выполнить проверки из всех затронутых локальных `AGENTS.md`.
- Обновить удалённый `master` через `git fetch origin master` и при необходимости безопасно перенести свою ветку на него. Не переписывать чужую историю и не использовать принудительную отправку без явного разрешения.
- Отправить рабочую ветку командой `git push -u origin <branch>`.

При создании Pull Request:

- Направлять его из рабочей ветки в `master`. Заголовок должен соответствовать Conventional Commits, быть короче 72 символов и описывать пользовательское изменение в повелительном наклонении, например `feat: add monitor creation`.
- В описании обязательно указать: пользовательский результат, ключевые технические изменения, выполненные проверки с их результатом, конфигурационные изменения, ограничения или отложенные проверки, ссылку на задачу/issue при наличии и скриншоты для визуальных frontend-изменений.
- После создания проверить base/head, заголовок, описание и ссылку на PR через `gh pr view` или веб-интерфейс, затем сообщить пользователю URL.
- Если отсутствуют `origin`, права на репозиторий или действующая авторизация GitHub, не выдумывать PR и не менять настройки доступа: сообщить точную причину и запросить подключение remote или повторную авторизацию.
## Validation

Before review, run the checks specified by the local `AGENTS.md` for every application changed. Avoid unrelated formatting or dependency updates.

## Коммиты

- Формат Conventional Commits (feat, fix, refactor, test, docs, chore);
- Заголовок до 72 символов, в повелительном наклонении;
- Без эмодзи, без "significantly improved" и прочей воды;
- Тело – только если нужно объяснять "почему", а не "что";
- Один логический шаг – один коммит.
