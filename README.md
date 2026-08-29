# Karl AI Agent Dotfiles

Configuracion global y portable para una topologia de desarrollo controlada:

```text
USER -> MAIN -> IMPLEMENTER -> MAIN -> VERIFIER -> MAIN
```

Solo MAIN coordina. IMPLEMENTER modifica el sistema. VERIFIER evalua el resultado de forma independiente y no repara. Los ciclos de reparacion se limitan a dos.

## Distribucion

| Contenido | OpenCode | Codex |
|---|---|---|
| `skills/*/SKILL.md` | Lee `~/.agents/skills` nativamente | Lee `~/.agents/skills` nativamente |
| Reglas de entrada | Bloque administrado en `~/.config/opencode/AGENTS.md` | Bloque administrado en `~/.codex/AGENTS.md` |
| Main | `karl-main.md`, agente primary | Sesion raiz guiada por AGENTS.md |
| Implementer | Markdown con modelo y permisos | TOML con modelo, esfuerzo y sandbox |
| Verifier | Markdown con modelo y permisos | TOML con `sandbox_mode = "workspace-write"` |

Las skills contienen los contratos portables. Los archivos bajo `harnesses/` contienen solo configuracion nativa: modelo, permisos, modo e instrucciones de entrada.

## Instalar

PowerShell 7 es requisito en Windows y Linux. En Windows, activa Developer Mode para crear symlinks sin privilegios. En Linux, el instalador usa el soporte normal de symlinks y no consulta el registro de Windows.

```powershell
pwsh -File "$HOME/.agents/install.ps1" -DryRun
pwsh -File "$HOME/.agents/install.ps1"
```

El instalador:

1. Valida las skills `karl-*`.
2. Inserta o actualiza solo el bloque delimitado por comentarios `karl-ai` en cada `AGENTS.md` global.
3. Crea symlinks por archivo para los custom agents de cada harness.
4. Conserva backups bajo `~/.agents-backup/<timestamp>/` antes de reemplazar contenido.

Si un custom agent ya existe y no es el symlink esperado, el instalador se detiene. `-Force` crea un backup y lo reemplaza.

## Desinstalar

```powershell
pwsh -File "$HOME/.agents/install.ps1" -Uninstall -DryRun
pwsh -File "$HOME/.agents/install.ps1" -Uninstall
```

La desinstalacion elimina solo:

- bloques delimitados por `<!-- karl-ai: controlled-development -->`
- symlinks cuyo destino esta dentro de este repositorio

No elimina otras reglas, skills o agentes.

## Modelos

OpenCode usa el proveedor oficial OpenCode Go. Sus IDs de modelo usan el formato `opencode-go/<model-id>`.

| Rol | OpenCode | Codex |
|---|---|---|
| Main | `opencode-go/glm-5.3-flash`, variant `high` | Modelo de la sesion raiz |
| Implementer | `opencode-go/glm-5.3-flash`, variant `high` | `gpt-5.6-terra`, effort `high` |
| Verifier | `opencode-go/glm-5.3-flash`, variant `high` | `gpt-5.6-sol`, effort `high` |

El verifier de Codex usa `workspace-write` para ejecutar suites que generan caches o artefactos. Su contrato prohibe editar codigo fuente, reparar hallazgos o dejar cambios persistentes.

## Sincronizar Otra Maquina

```powershell
git clone <private-repository> "$HOME/.agents"
pwsh -File "$HOME/.agents/install.ps1"
```

Instala o actualiza skills directamente dentro de `~/.agents/skills`; ambos harnesses las descubren sin pasos adicionales.

Las skills externas instaladas por otros gestores permanecen locales y no forman parte de este repositorio. Git solo versiona las skills Karl mantenidas aqui.

## Diagnostico De Skills

OpenCode no carga `~/.agents/skills` cuando se inicia con:

```text
OPENCODE_DISABLE_EXTERNAL_SKILLS=1
```

El instalador muestra un warning si detecta esa variable. Eliminala del entorno antes de iniciar OpenCode; no hace falta duplicar ni enlazar las skills dentro de `~/.config/opencode/skills`.

## E2E

La prueba offline ejecuta el ciclo completo en un home temporal. No requiere credenciales, Pester ni otras dependencias:

```powershell
pwsh -NoProfile -File tests/e2e/install-lifecycle.ps1
```

Desde Windows, tambien puedes reproducir la ejecucion Linux con Docker:

```powershell
docker run --rm --mount "type=bind,source=$PWD,target=/repo" -w /repo mcr.microsoft.com/powershell:latest pwsh -NoProfile -File tests/e2e/install-lifecycle.ps1
```

La prueba live requiere `OPENCODE_API_KEY`. Usa un home y un repositorio git temporales, ejecuta `karl-main` de forma no interactiva y conserva stdout JSON y stderr bajo `TestResults/`. La politica es instalar `opencode-ai@latest`; actualmente npm reporta la version `1.18.25`:

```powershell
npm install --global opencode-ai@latest
opencode --version
$env:OPENCODE_API_KEY = "<secret>"
pwsh -NoProfile -File tests/e2e/opencode-minimal.ps1
```

Configura `OPENCODE_API_KEY` como un Actions repository secret para habilitar el job live. El workflow ejecuta la prueba offline en una matriz de `windows-latest` y `ubuntu-latest` en cada `push`, `pull_request`, ejecucion manual y programada. La prueba live usa la misma matriz, se ejecuta solo de forma manual o programada, despues de la prueba offline y cuando existe el secret. Nunca se ejecuta en pull requests.

Los logs de Actions se publican como artifacts de diagnostico. `TestResults/` tambien contiene los diagnosticos de una ejecucion live local y esta excluido de Git.

La prueba live no se ejecuta en cada PR porque consume una API con credenciales y costo, depende de un servicio externo y puede introducir fallos transitorios que no representan una regresion del instalador.
