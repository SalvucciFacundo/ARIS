# MUSE — Motor Unificado de Síntesis Multimodal

## Spec de Proyecto (SDD Phase: Proposal + Design)

> **Versión:** 0.1-draft
> **Fecha:** 2026-08-07
> **Autor:** Facundo + ARIA
> **Nombre:** MUSE — Multimodal Unified Synthesis Engine (la musa / meditar)

---

## 1. Visión

Un agente de chat autónomo, con memoria y skills (inspirado en GAIA), que convierte
descripciones en lenguaje natural en **imágenes de alta calidad**, generando
automáticamente el prompt técnico completo (positivo, negativo, modelo, sampler,
steps, CFG, seed) y delegando la generación al backend más adecuado.

```
Usuario: "quiero una imagen de un perro saltando en la luna"
Agente:  [analiza → genera prompt técnico → llama backend → entrega imagen]
         + aprende preferencias del usuario con cada uso
```

## 2. Objetivos

- **Principal:** Que el usuario describa una imagen en lenguaje natural y reciba la imagen generada.
- **Secundario:** Que el agente genere prompts técnicos de calidad sin que el usuario sepa de SD/ComfyUI.
- **Diferenciador:** Memoria persistente — el agente recuerda estilos, temas y correcciones del usuario.
- **Extensible:** Backend intercambiable (Pollinations gratis → ComfyUI local → API paga).

## 3. No-Objetivos (fuera de alcance para MVP)

- ❌ No UI web completa (arranca por terminal/Telegram)
- ❌ No edición/outpainting/inpainting (fase 2)
- ❌ No entrenamiento de modelos propios
- ❌ No generación de video

## 4. Arquitectura

```
┌──────────────────────────────────────────────────────┐
│                 MUSE (Desktop App)                   │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │  UI — Wails (Go + HTML/CSS/JS vanilla)       │   │
│  │  • chat lateral (como ChatGPT)              │   │
│  │  • canvas de imagen (generada / subida)     │   │
│  │  • drag & drop para subir imagen            │   │
│  │  • sliders: steps, CFG, seed, modelo        │   │
│  │  • galería de historial                     │   │
│  └───────────┬──────────────────────────────────┘   │
│              │ (binding Go ↔ JS)                     │
│  ┌───────────▼────────────┐  ┌───────────────────┐  │
│  │  ORCHESTRATOR          │  │  MEMORY           │  │
│  │  • interpreta pedidos  │  │  • preferencias   │  │
│  │  • decide qué generar  │  │  • historial      │  │
│  │  • itera con el user   │  └───────────────────┘  │
│  └───────────┬────────────┘                         │
│              │                                       │
│  ┌───────────▼──────────────────────────┐           │
│  │  PROMPT ENGINE (LLM-powered)        │           │
│  │  • entiende el pedido en natural    │           │
│  │  • genera prompt positivo/negativo  │           │
│  │  • elige modelo/params              │           │
│  │  • multi-provider LLM:              │           │
│  │    OpenAI · Anthropic · Ollama      │           │
│  └───────────┬──────────────────────────┘           │
│              │                                       │
│  ┌───────────▼────────────┐                         │
│  │  IMAGE BACKEND (adapter)│                        │
│  │  • Pollinations (free) │                         │
│  │  • ComfyUI local       │                         │
│  │  • Stability/OpenAI API│                         │
│  └────────────────────────┘                         │
│                                                      │
│  GATEWAYS (fase 2, opcionales): Telegram | Discord  │
│  — el mismo core, otra capa de entrada               │
└──────────────────────────────────────────────────────┘
```

**Dos cerebros separados e intercambiables:**
- **LLM (razonamiento):** entiende lo que pedís y arma los prompts. Usa el proveedor que tengas configurado (OpenAI, Anthropic, Ollama local — el mismo patrón multi-provider de GAIA).
- **Image Backend (generación):** crea la imagen. Proveedor intercambiable: Pollinations gratis (default), ComfyUI local, o API paga (Stability/OpenAI/Replicate).

## 5. Stack Tecnológico

| Capa | Tecnología | Justificación |
|---|---|---|
| Lenguaje | **Go 1.22+** | Mismo stack que GAIA, binario único |
| UI Desktop | **Wails v2** (Go + HTML/CSS/JS) | 35.7K stars, activo, cross-platform |
| Frontend | **Vanilla HTML + Tailwind + JS** | Sin framework — menos piezas, más liviano |
| LLM (razonamiento) | **Multi-provider** (OpenAI, Anthropic, Ollama) | Entiende pedidos y arma prompts — como GAIA |
| Memoria | SQLite (modernc.org/sqlite) + JSON | Cero deps externas, persistente |
| Image Backend 1 | **Pollinations.ai API** (gratis, sin key) | VERIFICADO: 512x512 en ~1s |
| Image Backend 2 | ComfyUI local | Sin costo, para usuarios con GPU |
| Image Backend 3 | Stability / OpenAI / Replicate API | Pago, alta calidad |
| Gateways | Telegram/Discord (fase 2) | El mismo core, otra capa de entrada |
| Config | YAML/TOML + env | Multi-provider LLM como GAIA |

## 6. API de Pollinations (verificada hoy)

```
GET https://image.pollinations.ai/prompt/{prompt}?width=512&height=512&seed=42&nologo=true&model=flux
```

| Parámetro | Valores | Uso |
|---|---|---|
| `prompt` | texto URL-encoded | Prompt positivo |
| `width/height` | 256-1024+ | Resolución |
| `seed` | int | Reproducibilidad |
| `nologo` | true/false | Quitar watermark |
| `model` | flux, turbo, etc. | Modelo de generación |
| `negative` | texto (verificar) | Prompt negativo |

**Resultado verificado:** HTTP 200, JPEG real, 512x512 en 1.0-1.9s, sin API key.

## 7. Flujo de Usuario (MVP)

```
$ muse
🧠 MUSE: Hola! Decime qué imagen querés crear.
Tú> un perro saltando en la luna
🧠 MUSE: [pensando...] Genero con modelo flux, 512x512.
   Prompt: "a happy dog jumping high on the moon surface, earth visible in the background,
            astronaut helmet, stars, cinematic lighting, ultra detailed"
   Negativo: "blurry, low quality, distorted, extra limbs, watermark"
   ✓ Imagen guardada en ./output/perro_luna_001.png
   ¿Te gusta? (sí / más oscura / sin el casco / etc.)
Tú> más oscura y sin casco
🧠 MUSE: [recuerda tu preferencia: "sin casco", "fondo oscuro"]
   ✓ Imagen regenerada con seed nuevo
```

## 8. Requisitos Funcionales (MVP)

### RF-1: Chat conversacional
- [ ] El usuario escribe una descripción en lenguaje natural
- [ ] El agente responde en el mismo idioma del usuario (ES/EN)
- [ ] Soporta aclaraciones iterativas ("más oscura", "sin el perro")

### RF-2: Prompt Engine (LLM-powered)
- [ ] Convierte la descripción del usuario en prompt positivo detallado en inglés
- [ ] Genera prompt negativo automático (calidad, artefactos, anatomía)
- [ ] Elige parámetros: modelo de imagen, width/height, seed — según el pedido y las preferencias
- [ ] Multi-provider LLM: funciona con OpenAI, Anthropic u Ollama local (configurable)
- [ ] Expone los prompts al usuario para que los vea/edite

### RF-3: Generación
- [ ] Llama a Pollinations con los parámetros
- [ ] Guarda la imagen en `./output/YYYYMMDD_HHMMSS_desc.png`
- [ ] Muestra preview (terminal: ruta + tamaño; Telegram: imagen directa)

### RF-4: Memoria
- [ ] Guarda preferencias de estilo del usuario (historial de correcciones)
- [ ] Incorpora preferencias al prompt en generaciones futuras
- [ ] Historial de imágenes generadas (prompt + seed + params)

### RF-5: Iteración
- [ ] El usuario puede pedir cambios ("más oscura") → se regenera
- [ ] El seed cambia en cada iteración salvo que el usuario pida lo mismo

## 9. Requisitos No-Funcionales

- **Rendimiento:** < 5s por imagen (Pollinations), < 1s para el análisis del prompt
- **Portabilidad:** binario único, funciona en Linux/macOS/Windows
- **Offline-friendly:** el agente funciona sin GPU; solo requiere internet para Pollinations
- **Seguridad:** no ejecuta prompts como shell; sanitiza entrada
- **Config:** via archivo de config (modelo default, backend default, carpeta output)

## 10. Estructura de Archivos Propuesta

```
muse/
├── go.mod
├── README.md
├── cmd/
│   └── muse/
│       └── main.go              # entrypoint Wails app
├── internal/
│   ├── agent/
│   │   ├── orchestrator.go      # loop principal, interpreta pedidos
│   │   └── conversation.go      # estado de conversación
│   ├── prompt/
│   │   ├── engine.go            # descripción → prompt positivo/negativo
│   │   ├── engine_test.go
│   │   └── params.go            # modelo, steps, cfg, seed
│   ├── backend/
│   │   ├── backend.go           # interfaz Backend
│   │   ├── pollinations.go      # adapter Pollinations
│   │   └── pollinations_test.go
│   ├── memory/
│   │   ├── store.go             # SQLite/JSON persistente
│   │   └── preferences.go       # preferencias de usuario
│   └── ui/
│       ├── app.go               # Wails bindings (métodos expuestos a JS)
│       └── frontend/            # Svelte/React + Tailwind
│           ├── src/App.svelte   # chat lateral + canvas + galería
│           └── src/...          
├── output/                      # imágenes generadas (gitignore)
└── config.yaml                  # modelo default, backend, etc.
```

**UI Desktop (Wails) — pantalla principal:**
- **Lateral izquierdo:** historial de conversaciones + galería de imágenes
- **Centro:** canvas de la imagen (generada o subida con drag & drop)
- **Derecha:** chat con MUSE (como ChatGPT) + panel de parámetros (sliders: steps, CFG, seed, modelo) + botón "Generar"

## 11. Plan de Implementación (TDD, tareas 2-5 min)

### Tarea 1: Setup del proyecto
- [ ] `go mod init muse`, estructura de carpetas
- [ ] Instalar Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- [ ] `wails init` con template Svelte/React + Tailwind
- [ ] Config básica (config.yaml + loader)

### Tarea 2: Backend Pollinations (TDD)
- [ ] Test: `TestBuildPromptURL` — prompt + params → URL correcta
- [ ] Test: `TestFetchImage` — HTTP 200 y JPEG válido (mock del server)
- [ ] Implementar `pollinations.go` (GET a image.pollinations.ai, guarda archivo)

### Tarea 3: Prompt Engine con LLM (TDD)
- [ ] Test: `TestLLMProviderInterface` — interfaz Provider con `Complete(prompt) → string`
- [ ] Test: `TestBuildPositivePrompt` — dado el pedido del usuario, arma prompt en inglés (mock del LLM)
- [ ] Test: `TestBuildNegativePrompt` — siempre incluye artefactos básicos + lo que el LLM agregue
- [ ] Test: `TestPickParams` — defaults correctos (flux, 512, seed random)
- [ ] Implementar `engine.go`: llama al LLM con un system prompt de "prompt engineer de imágenes" y parsea la respuesta (JSON: prompt, negative, model, params)
- [ ] Implementar providers: `openai.go`, `anthropic.go`, `ollama.go` (interfaz común, como GAIA)

### Tarea 4: Memoria (TDD)
- [ ] Test: `TestSaveLoadPreferences` — persiste y recarga
- [ ] Test: `TestApplyPreferencesToPrompt` — las preferencias modifican el prompt
- [ ] Implementar `store.go` + `preferences.go` (JSON file, simple)

### Tarea 5: Orquestador (TDD)
- [ ] Test: `TestInterpretRequest` — detecta "generar", "cambiar", "salir"
- [ ] Test: `TestIterationFlow` — "más oscura" regenera con nuevo prompt
- [ ] Test: `TestEditImageRequest` — detecta "modificar esta imagen" (img2img)
- [ ] Implementar `orchestrator.go` (loop simple: leer → interpretar → generar → guardar → responder)

### Tarea 6: Wails bindings (Go ↔ JS)
- [ ] `app.go` con métodos expuestos: `GenerateImage`, `EditImage`, `GetHistory`, `SavePreferences`
- [ ] Test de bindings (los métodos llaman al core y devuelven resultados)

### Tarea 7: Frontend desktop (UI)
- [ ] Layout 3 paneles: historial | canvas | chat+params
- [ ] Chat funcional: escribir descripción → MUSE responde → imagen aparece en canvas
- [ ] Panel de parámetros: sliders steps/CFG/seed, selector de modelo, botón Generar
- [ ] **Drag & drop**: subir imagen → aparece en canvas → botón "Modificar" la envía al backend (img2img vía Pollinations)
- [ ] Galería: historial de imágenes + clic para volver a verla
- [ ] Dark premium: gradientes teal→emerald, glassmorphism, fondo #08080c (tu estilo)

### Tarea 8: Gateways (fase 2, opcional)
- [ ] Bot de Telegram que recibe "imagen de X" y responde con la imagen
- [ ] Bot de Discord (mismo core, otro adapter)

### Tarea 9: Pulido y release
- [ ] README con ejemplos y capturas
- [ ] Tests completos pasando
- [ ] Release v0.1.0 (binarios Windows/macOS/Linux vía Wails)

## 12. Verificación / Aceptación

1. `go test ./...` → todo verde
2. E2E: `echo "un perro saltando en la luna" | muse` → genera imagen en output/
3. E2E Telegram: enviar "imagen de gato astronauta" → responde con imagen
4. Memoria: pedir "más oscuro" dos veces → tercera generación ya sale oscura por defecto

## 13. Riesgos y Decisiones Abiertas

| Riesgo | Mitigación |
|---|---|
| Pollinations cambia/limita la API gratis | Adapter intercambiable; fase 2 ComfyUI local |
| Calidad de imagen variable | Modelo flux por default; exponer parámetros |
| Prompt sin LLM = limitado | Fase 2: conectar LLM (Ollama local o API) para prompts ricos |
| ARM64 sin GPU | Pollinations es cloud — no afecta; ComfyUI local sería lento (fase 2 opcional) |

**Decisiones pendientes (para Facundo):**
1. ¿Nombre final del proyecto? (IMAGEN / muse / otro)
2. ¿MVP por terminal (TUI) o ya con Telegram?
3. ¿Prompt Engine con heurística primero (sin LLM) o directamente con LLM local/API?
4. ¿Backend default: Pollinations gratis confirmado?

## 14. Roadmap

- **MVP (semana 1):** Tareas 1-6 → terminal, Pollinations, memoria básica, iteración
- **Fase 2 (semana 2):** Telegram, LLM para prompts ricos, ComfyUI local opcional
- **Fase 3:** Web UI, inpainting/outpainting, modelos múltiples
