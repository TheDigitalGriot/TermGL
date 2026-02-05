# TermGL

A 3D graphics rendering library for terminal user interfaces in Go. Built for the [Charm](https://charm.sh) ecosystem.

```
                            @::%@#%%%%%#@@@##+%%%%%%%%%*##%%#-##+++*#***#
                            @::@@#%%%+%#-@@+%++%%%%%%%**@-%%=-#:+++*##**#
                            @:.@@#%%%%%@@@+%*##+%%%%%***%@-######++*##+*#
                            @:.@@%%%%%%%@#@%%##++%%%****@@###+###++*##:*#
                            @:@@@%%%%%%@@%@*@##+++%*****@%#@#####++*###*#
                            @@+.@%%%@@@@@%@*@##+++%*****@%#@#####++*#:-##
                            @##.@%#%@%@@@%@%###+++*******@#@###*#+=*#:**#
                             *.+@%.%@@@@@%%@###+++*******@@@#####+:*#-:=
                             ##+.%.%@@%@@%@@###+++*******@@@##*##+:*:-**
                             @*#.#.%@@@@@@@###++++%%******@@#####+:=:*=*
                              @..:..@@%@@@@###++++%%******@@##*##::.::*
                              @*#:+-@@%%%@####++++%********@***##-=.*=*
                               @.===@@@%@@####++++%%%******@#*###::::*
                               @@%%=#@@%%@######++%%%******@**##+:++**
                                @%%***+@%%######++%%*********#=###***
```
*Suzanne rendered in the terminal at 30fps*

## Features

- **4-Layer Architecture**: Canvas → 2D Primitives → 3D Pipeline → Animation
- **Full 3D Pipeline**: Perspective projection, scanline rasterization, Z-buffering
- **ASCII Shading**: Flat and smooth (Gouraud) shading with configurable luminance ramps
- **OBJ Loading**: Load 3D models from Wavefront OBJ files
- **Bubble Tea Integration**: Use as a standard `tea.Model` component
- **Harmonica Springs**: Physics-based animation for smooth motion
- **Pure Go**: No CGo, cross-compiles everywhere Go runs

## Installation

```bash
go get github.com/charmbracelet/termgl
```

## Quick Start

### Spinning Cube

```go
package main

import (
    "fmt"
    "os"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/termgl/anim"
    "github.com/charmbracelet/termgl/canvas"
    "github.com/charmbracelet/termgl/gl"
    "github.com/charmbracelet/termgl/math"
)

type model struct {
    canvas   *canvas.Canvas
    renderer *gl.Renderer
    cube     *gl.Mesh
    angle    float64
}

type tickMsg time.Time

func (m model) Init() tea.Cmd {
    return tea.Tick(time.Second/30, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "q" {
            return m, tea.Quit
        }
    case tickMsg:
        m.angle += 1.5
        m.cube.SetRotation(m.angle*0.3, m.angle, 0)
        return m, tea.Tick(time.Second/30, func(t time.Time) tea.Msg {
            return tickMsg(t)
        })
    }
    return m, nil
}

func (m model) View() string {
    m.renderer.Clear()
    m.renderer.RenderMesh(m.cube)
    return m.canvas.String()
}

func main() {
    // Create canvas and camera
    c := canvas.New(80, 24)
    cam := gl.NewPerspectiveCamera(30, c.Aspect(), 0.1, 1000)
    cam.SetPosition(0, 0, 6)
    cam.LookAt(math.Vec3{})

    // Create renderer with lighting
    r := gl.NewRenderer(c, cam)
    r.SetDirectionalLight(gl.NewDirectionalLight(math.Vec3{X: 0.5, Y: 0.5, Z: -1}, 1.0))
    r.SetAmbientLight(gl.NewAmbientLight(0.2))

    // Create cube
    cube := gl.NewCube()

    m := model{canvas: c, renderer: r, cube: cube}

    if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
        fmt.Println("Error:", err)
        os.Exit(1)
    }
}
```

### Loading an OBJ Model

```go
// Load a model
mesh, err := gl.LoadOBJFile("model.obj")
if err != nil {
    log.Fatal(err)
}

// Position and render
mesh.SetRotation(0, 45, 0)
renderer.RenderMesh(mesh)
```

### Spring-Animated Properties

```go
// Create animated rotation with Harmonica springs
rotY := anim.NewAnimatedFloat(0, 5.0, 0.5, 30) // initial, frequency, damping, fps

// Set target - spring will animate smoothly
rotY.Set(90)

// In update loop
rotY.Update()
mesh.SetRotation(0, rotY.Get(), 0)
```

## Architecture

```
termgl/
├── math/      # Linear algebra (Vec2, Vec3, Vec4, Mat4, Transform)
├── canvas/    # Terminal framebuffer with Z-buffer
├── draw/      # 2D primitives (lines, shapes)
├── gl/        # 3D rendering pipeline
│   ├── mesh.go        # Vertex, Triangle, Mesh types
│   ├── primitives.go  # Built-in Cube, Plane, Pyramid
│   ├── obj.go         # OBJ file loader
│   ├── camera.go      # Perspective/orthographic camera
│   ├── light.go       # Directional + ambient lighting
│   ├── rasterizer.go  # Scanline triangle rasterization
│   ├── shader.go      # Flat + smooth shading
│   └── renderer.go    # Rendering orchestration
└── anim/      # Bubble Tea + Harmonica integration
    ├── viewport.go    # tea.Model wrapper
    └── animated.go    # Spring-animated properties
```

## Render Modes

```go
renderer.RenderMode = gl.RenderShaded    // Lit with ASCII shading (default)
renderer.RenderMode = gl.RenderWireframe // Edge lines only
renderer.RenderMode = gl.RenderSolid     // Single character fill
renderer.RenderMode = gl.RenderOutlined  // Wireframe + solid
renderer.RenderMode = gl.RenderDepth     // Z-buffer visualization
```

## Shading Modes

```go
renderer.ShadingMode = gl.ShadingFlat   // One shade per face (default)
renderer.ShadingMode = gl.ShadingSmooth // Interpolated vertex normals (Gouraud)
```

## Luminance Ramp

The default character gradient for shading:

```
 .:-=+*#%@
```

Customize with:

```go
renderer.SetLuminanceRamp(" .,;:!vlLFE$")
```

## Examples

Run the included examples:

```bash
# Spinning cube with keyboard controls
go run ./examples/cube

# Suzanne model viewer
go run ./examples/suzanne
```

**Controls:**
- Arrow keys: Rotate model
- Space: Toggle auto-rotation
- 1-5: Change render mode
- +/-: Zoom (Suzanne only)
- Q: Quit

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Harmonica](https://github.com/charmbracelet/harmonica) - Spring physics

## Performance

Targets 30+ fps with:
- 200×60 cell canvas
- 1K triangle mesh
- ANSI backend

## License

MIT License - see LICENSE file

## Acknowledgments

- Rendering algorithms ported from [ascii-graphics](https://github.com/addr0x414b/ascii-graphics)
- Animation patterns from [Harmonica](https://github.com/charmbracelet/harmonica)
- Inspired by the [Charm](https://charm.sh) ecosystem
