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

1. Publicar release `v0.2.0` en GitHub.
2. Registrar release en `emir-core` con `is_mandatory = true`.
3. Reiniciar el agente o esperar el ciclo de 10 min.
4. Verificar que el servicio se detiene, el binario se reemplaza y el servicio vuelve a iniciar.

### 4. Plataformas

Repetir las pruebas anteriores en:

- Windows 10/11
- Ubuntu 22.04/24.04
- macOS 13+
