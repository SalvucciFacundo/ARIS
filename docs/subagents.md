# ARIS Specialized Visual Subagents

ARIS includes a suite of specialized AI visual subagents that act as expert virtual art directors, prompt engineers, and photographers. You can target any subagent directly using the `@<name>` syntax across all interfaces (CLI, TUI, Web GUI, Telegram, and Discord).

---

## Subagent Catalog

| Subagent | Role | Trigger Syntax | Specialty |
|---|---|---|---|
| `@director` | Art Director & Master Stylist | `aris gen "@director <prompt>"` | High-concept cinematography, lighting composition, spatial harmony, and visual mood. |
| `@promptsmith` | Prompt Optimization Engineer | `aris gen "@promptsmith <prompt>"` | Technical keyword conversion, negative trigger curation, and model-specific weighting. |
| `@photorealism` | Optical & Camera Specialist | `aris gen "@photorealism <prompt>"` | Focal length (35mm, 85mm), aperture (f/1.4), bokeh, shutter speed, ISO, and realistic skin texture. |
| `@anime` | Japanese Animation Director | `aris gen "@anime <prompt>"` | Makoto Shinkai / Studio Ghibli cel-shading, vibrant saturated palettes, anime lineart. |
| `@concept-art` | Production Concept Artist | `aris gen "@concept-art <prompt>"` | Worldbuilding, matte painting, sci-fi/fantasy environment staging, and vehicle design. |
| `@cyberpunk` | Neo-Tokyo & Sci-Fi Specialist | `aris gen "@cyberpunk <prompt>"` | Volumetric neon lighting, holograms, chrome reflections, rain-soaked pavement. |
| `@pixelart` | Retro Pixel Specialist | `aris gen "@pixelart <prompt>"` | Isometric pixel art, 16-bit palettes, clean dithering, sprite sheets. |
| `@critic` | Visual Critic & Evaluator | `aris subagents run critic "<notes>"` | Anatomical accuracy, lighting consistency, and prompt constraint verification. |
| `@inpainter` | Inpainting & Blending Artist | `aris edit <img.png> "@inpainter <prompt>"` | Seamless mask blending, object removal, and seamless inpainting repairs. |
| `@restyler` | Style Transfer Specialist | `aris edit <img.png> "@restyler <prompt>"` | Reference image restyling with calibrated denoise strength defaults (`0.65`). |

---

## Subagent Customization & Persistence

Subagents are persisted in the SQLite database and can be customized at runtime or in code:

```bash
# List all active subagents
aris subagents list

# Inspect system prompt and configuration of a subagent
aris subagents show director

# Run a subagent prompt directly
aris subagents run anime "cyberpunk motorcycle racer in neo tokyo"
```

---

## Defining Custom Subagents in Go

Custom subagents can be added to the registry via `SubagentStore`:

```go
customSubagent := domain.SubagentDef{
    Name:         "steampunk",
    DisplayName:  "Steampunk Architect",
    Role:         "Victorian Sci-Fi Specialist",
    SystemPrompt: "You are an expert in Steampunk aesthetic. Focus on brass gears, copper pipes, steam plumes, Victorian attire, and warm sepia/amber lighting.",
    Temperature:  0.7,
    AllowedTools: []string{"reasoner", "prompt_architect"},
}

subagentManager.SaveSubagent(ctx, customSubagent)
```
