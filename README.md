# EMIR Agent

Agente de escritorio multiplataforma para el sistema EMIR. Se instala en cada PC del inventario, se vincula manualmente con el backend y reporta periódicamente hardware, software y estado de red.

- **Backend:** [emir-core](https://github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-core) (FastAPI)
- **Lenguaje:** Go 1.22+
- **Repositorio:** https://github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent

## Características

- **Comunicación saliente:** solo el agente llama al backend; no recibe conexiones entrantes.
- **Vinculación manual:** el técnico ingresa URL del core y un código de emparejamiento de un solo uso.
- **Autenticación segura:** token + firma Ed25519 por request.
- **Almacenamiento seguro:** token y clave privada en el keyring del sistema operativo.
- **Inventario multiplataforma:** recolecta CPU, RAM, discos, GPUs, monitores, red, periféricos, activación de Windows/Office, antivirus, Wazuh y RustDesk.
- **Auto-update:** detecta nuevas versiones publicadas en GitHub Releases y se actualiza automáticamente cuando el backend marca la versión como obligatoria.

## Arquitectura

```
┌─────────────┐     POST /api/agent/pair        ┌─────────────┐
│   Técnico   │ ──────────────────────────────▶ │  emir-core  │
│  (consola)  │                                 │  (FastAPI)  │
└──────┬──────┘                                 └──────┬──────┘
       │                                               │
       │ POST /api/agent/heartbeat                     │
       │ POST /api/agent/inventory  (cada 10 min)      │
       │ GET  /api/agent/version                       │
       ◀───────────────────────────────────────────────┘
```

## Requisitos

- Go 1.22+ (solo para desarrollo/build).
- Privilegios de administrador/root para la instalación como servicio.
- Conexión al backend `emir-core`.

## Instalación

### Windows (PowerShell como Administrador)

```powershell
Invoke-Expression "& { $(Invoke-RestMethod https://github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/releases/download/v0.1.0/install-windows.ps1) } -CoreURL https://core.emir.example.com -Version 0.1.0"
```

O descarga y ejecuta manualmente:

```powershell
.\scripts\install-windows.ps1 -CoreURL "https://core.emir.example.com" -Version "0.1.0"
```

El script:
1. Descarga el release.
2. Instala el binario en `C:\Program Files\emir-agent\emir-agent.exe`.
3. Ejecuta `emir-agent.exe --pair` para vincular interactivamente.
4. Registra e inicia el servicio `emir-agent` con `sc.exe`.

### Linux (como root)

```bash
curl -fsSL https://github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/releases/download/v0.1.0/install-linux.sh | bash -s "https://core.emir.example.com" "0.1.0"
```

O localmente:

```bash
bash scripts/install-linux.sh "https://core.emir.example.com" "0.1.0"
```

### macOS (como root)

```bash
curl -fsSL https://github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/releases/download/v0.1.0/install-macos.sh | bash -s "https://core.emir.example.com" "0.1.0"
```

O localmente:

```bash
bash scripts/install-macos.sh "https://core.emir.example.com" "0.1.0"
```

## Configuración

El agente se configura por variables de entorno:

| Variable | Descripción | Default |
|---|---|---|
| `EMIR_CORE_URL` | URL base del backend EMIR | `http://localhost:8000` |
| `EMIR_STATE_PATH` | Ruta del `state.json` local | Directorio de config del usuario |

En Windows el instalador guarda `EMIR_CORE_URL` como variable de entorno de máquina. En Linux se escribe en `/etc/default/emir-agent`.

## Vinculación manual

El agente debe estar vinculado a un `Computer` existente en `emir-core`. El flujo:

1. En el frontend de EMIR, abrir el detalle del equipo y hacer clic en **"Vincular equipo"**.
2. El backend genera un código de vinculación de un solo uso.
3. En la PC, ejecutar el agente en modo par:

```powershell
# Windows
C:\Program Files\emir-agent\emir-agent.exe --pair

# Linux
/opt/emir-agent/emir-agent --pair

# macOS
/usr/local/bin/emir-agent --pair
```

4. Ingresar la URL del core y el código. El agente guarda el token y la clave privada de forma segura.

## Actualización automática

1. Publica una nueva versión del agente:

```bash
git tag v0.2.0
git push origin v0.2.0
```

2. GitHub Actions genera los assets por plataforma.
3. Registra la versión en `emir-core` (`agent_releases`):

```sql
INSERT INTO agent_releases (
    id, version,
    download_url, checksum,
    linux_download_url, linux_checksum,
    macos_download_url, macos_checksum,
    is_mandatory, release_notes, enabled, active,
    registration_date, last_update
) VALUES (
    gen_random_uuid(),
    '0.2.0',
    'https://github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/releases/download/v0.2.0/emir-agent-windows-amd64.exe',
    '<sha256-windows>',
    'https://github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/releases/download/v0.2.0/emir-agent-linux-amd64',
    '<sha256-linux>',
    'https://github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/releases/download/v0.2.0/emir-agent-macos-amd64',
    '<sha256-macos>',
    true,
    'Notas de la versión',
    true, true,
    now(), now()
);
```

4. En el siguiente ciclo de polling, el agente detecta la nueva versión, descarga el asset correspondiente a su plataforma, verifica el checksum y se reemplaza a sí mismo reiniciando el servicio.

## Desarrollo

```bash
# Clonar
git clone https://github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent.git
cd emir-agent

# Dependencias
go mod tidy

# Ejecutar en modo vinculación
go run . --pair

# Ejecutar ciclo normal
go run .

# Compilar
go build -o emir-agent .

# Cross-compile
GOOS=windows GOARCH=amd64 go build -o emir-agent.exe .
GOOS=linux   GOARCH=amd64 go build -o emir-agent .
GOOS=darwin  GOARCH=amd64 go build -o emir-agent .
```

## Build de releases

```bash
bash scripts/build-all.sh 0.2.0 dist
```

Genera en `dist/`:

- `emir-agent-windows-amd64.exe` + `.zip` + `.sha256`
- `emir-agent-linux-amd64` + `.tar.gz` + `.sha256`
- `emir-agent-macos-amd64` + `.tar.gz` + `.sha256`

## Seguridad

- Clave privada Ed25519 almacenada en el keyring del SO.
- Cada request firmado con timestamp para evitar replay.
- Checksum SHA256 verificado antes de reemplazar el binario en auto-update.
- Comunicación por HTTPS en producción.

## Licencia

Proyecto privado — Subsecretaría de TIC de Santa Rosa de Cabal.
