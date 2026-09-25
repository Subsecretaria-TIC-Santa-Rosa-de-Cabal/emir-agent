# Pruebas de EMIR Agent

## Tests unitarios

Ejecutar desde `emir-agent/`:

```bash
go test ./...
```

### Tests incluidos

- `internal/auth/auth_test.go`
  - Generación de par de claves Ed25519.
  - Codificación base64 de la clave pública.
  - Firma de timestamp.
  - Reconstrucción del par de claves desde seed o clave privada completa.
  - Manejo de claves privadas inválidas.

- `internal/models/models_test.go`
  - Selección de asset correcto por plataforma (`windows`, `linux`, `darwin`).
  - Manejo de plataformas no soportadas.
  - Manejo de assets faltantes para una plataforma.

- `internal/updater/updater_test.go`
  - Verificación de checksum SHA256 válido.
  - Verificación de checksum SHA256 inválido.
  - Manejo de asset no disponible para la plataforma actual.

## Pruebas de integración manual

### Requisitos

- `emir-core` corriendo y accesible.
- Un `Computer` registrado en el backend.
- Acceso de administrador en la PC donde se instalará el agente.

### 0. Verificar versión compilada

```powershell
# Windows
.\emir-agent.exe --version
```

Debe imprimir la versión inyectada por el release (por ejemplo `0.3.3`), no `0.1.0`.

### 1. Vinculación

```powershell
# Windows
.\emir-agent.exe --pair
```

Ingresar URL del core y código de vinculación generado desde el frontend.

### 2. Ciclo de vida

```powershell
# Instalar como servicio y arrancar
.\scripts\install-windows.ps1 -CoreURL "https://core.example.com" -Version "0.1.0"
```

Verificar logs o consola:
- `heartbeat OK`
- `inventory synced`
- `new agent version available` (si aplica)

### 3. Auto-update

#### Local (sin publicar en GitHub)

```powershell
# Windows
.\scripts\test-update.ps1 -CoreURL http://localhost:8000 -OldVersion 0.3.0 -NewVersion 0.3.5
```

```bash
# Linux / macOS
bash scripts/test-update.sh http://localhost:8000 0.3.0 0.3.5
```

El script:
1. Construye dos binarios (versión vieja y nueva).
2. Levanta un servidor HTTP local con el binario nuevo.
3. Imprime el SQL para registrar la versión fake en `emir-core`.
4. Ejecuta el binario viejo para observar el update.

#### En producción

1. Publicar release `v0.2.0` en GitHub.
2. Registrar release en `emir-core` con `is_mandatory = true`.
3. Reiniciar el agente o esperar el ciclo de 10 min.
4. Verificar que el servicio se detiene, el binario se reemplaza y el servicio vuelve a iniciar.
5. Si falla en Windows, revisar `C:\Program Files\emir-agent\update.log`.

#### Protección contra bucles

El agente guarda la última versión a la que intentó actualizarse en `state.json`. Si un release se compila sin el `-ldflags` correcto y el binario reporta `0.1.0`, el agente usa la versión recordada en `state.json` para no volver a intentar la misma actualización indefinidamente.

### 4. Plataformas

Repetir las pruebas anteriores en:

- Windows 10/11
- Ubuntu 22.04/24.04
- macOS 13+
