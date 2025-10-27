# FlapGame API Specification

**Version:** 1.0.0
**Foundation:** SDL3
**Target:** 2D and 3D game development with minimal boilerplate

## Design Principles

1. **Unified 2D/3D** - Single consistent API for both dimensions
2. **Functional & Immutable** - State transformations, not mutations
3. **Zero Boilerplate** - Initialize in one line, run in three
4. **Built-in Everything** - Physics, particles, audio, animations included
5. **Declarative Scenes** - Define what you want, not how to build it
6. **Performance First** - Hardware acceleration, batching, culling automatic

---

## Core API

### Application Lifecycle

```flap
import game

// Minimal initialization
app := game.init("Window Title", 800, 600)

// With options
app := game.init("Window Title", 800, 600, {
    fullscreen: false,
    vsync: true,
    msaa: 4,
    target_fps: 60,
    resizable: true
})

// Automatic game loop
app.run(initial_state, update_fn, render_fn)

// Manual control
@ app.running() {
    events := app.poll_events()
    dt := app.delta_time()
    state <- update(state, events, dt)
    app.clear(0x000000)
    render(state, app)
    app.present()
}

// Cleanup (automatic on scope exit)
app.quit()
```

---

## 2D Graphics

### Immediate Mode Drawing

```flap
// Primitives
app.draw.circle(x, y, radius, color)
app.draw.rect(x, y, width, height, color)
app.draw.line(x1, y1, x2, y2, color, thickness)
app.draw.triangle(x1, y1, x2, y2, x3, y3, color)
app.draw.polygon(points, color)
app.draw.ellipse(x, y, rx, ry, color)

// Styled drawing
app.draw.circle(x, y, r, color, {
    filled: true,
    outline: 0xFFFFFF,
    outline_width: 2,
    shadow: {x: 2, y: 2, blur: 5, color: 0x000000}
})

app.draw.rect(x, y, w, h, color, {
    rounded: 10,           // Corner radius
    gradient: {            // Linear gradient
        start: 0xFF0000,
        end: 0x0000FF,
        angle: 90
    }
})

// Text rendering
app.draw.text(x, y, "Hello World", size, color)
app.draw.text(x, y, "Styled", size, color, {
    font: "Arial",
    bold: true,
    italic: false,
    align: "center",      // left, center, right
    shadow: true
})

// Textures
texture := app.load.texture("sprite.png")
app.draw.texture(texture, x, y)
app.draw.texture(texture, x, y, {
    scale: {x: 2.0, y: 2.0},
    rotation: 45,          // Degrees
    flip_h: false,
    flip_v: false,
    tint: 0xFF00FF,
    alpha: 0.5,
    origin: {x: 0.5, y: 0.5}  // Pivot point (0-1)
})

// Clipping regions
app.draw.push_clip(x, y, w, h)
app.draw.circle(...)  // Clipped to region
app.draw.pop_clip()
```

### Sprite System

```flap
// Create sprite
sprite := app.sprite.create("player.png", {
    x: 100, y: 100,
    layer: 1,              // Render order
    visible: true
})

// Sprite with physics
sprite := app.sprite.create("player.png", {
    x: 100, y: 100,
    physics: {
        enabled: true,
        body_type: "dynamic",  // dynamic, static, kinematic
        collider: "box",       // box, circle, polygon
        mass: 1.0,
        friction: 0.5,
        restitution: 0.2,      // Bounciness
        gravity_scale: 1.0
    }
})

// Sprite groups
enemies := app.group.create([enemy1, enemy2, enemy3])
enemies <- app.group.add(enemies, new_enemy)
enemies <- app.group.remove(enemies, dead_enemy)

// Batch operations
@ sprite in enemies {
    app.sprite.draw(sprite)
}

// Animation from spritesheet
animation := app.animation.create({
    texture: "player_sheet.png",
    frame_width: 32,
    frame_height: 32,
    frames: [0, 1, 2, 3, 4],
    fps: 10,
    loop: true
})

sprite <- app.sprite.set_animation(sprite, animation)
sprite <- app.sprite.play_animation(sprite, "walk")
```

### Particle System

```flap
// Create particle emitter
emitter := app.particles.create(x, y, {
    rate: 100,             // Particles per second
    lifetime: 2.0,         // Seconds
    velocity: {x: 0, y: -100},
    velocity_random: 50,   // +/- variance
    acceleration: {x: 0, y: 50},
    size_start: 10,
    size_end: 2,
    color_start: 0xFFFF00,
    color_end: 0xFF0000,
    alpha_start: 1.0,
    alpha_end: 0.0,
    rotation_speed: 180,   // Degrees per second
    blend_mode: "additive" // normal, additive, multiply
})

// Control
app.particles.start(emitter)
app.particles.stop(emitter)
app.particles.burst(emitter, count: 50)

// Update and render
app.particles.update(emitter, dt)
app.particles.draw(emitter)
```

---

## 3D Graphics

### Scene Management

```flap
// Declarative scene definition
scene := {
    camera: {
        type: "perspective",    // perspective, orthographic
        position: {x: 0, y: 5, z: 10},
        look_at: {x: 0, y: 0, z: 0},
        fov: 60,
        near: 0.1,
        far: 1000
    },
    lights: [
        {
            type: "directional",
            direction: {x: -1, y: -1, z: -1},
            color: 0xFFFFFF,
            intensity: 1.0
        },
        {
            type: "point",
            position: {x: 5, y: 5, z: 5},
            color: 0xFF0000,
            intensity: 0.5,
            range: 10.0,
            attenuation: {constant: 1.0, linear: 0.09, quadratic: 0.032}
        },
        {
            type: "ambient",
            color: 0x404040,
            intensity: 0.2
        }
    ],
    objects: [
        {
            type: "mesh",
            geometry: "cube",       // cube, sphere, plane, cylinder, cone
            position: {x: 0, y: 0, z: 0},
            rotation: {x: 0, y: 0, z: 0},
            scale: {x: 1, y: 1, z: 1},
            material: {
                type: "pbr",        // pbr, phong, unlit
                albedo: 0xFF0000,
                metallic: 0.5,
                roughness: 0.3,
                texture: "diffuse.png",
                normal_map: "normal.png"
            }
        },
        {
            type: "model",
            path: "assets/player.obj",
            position: {x: 5, y: 0, z: 0},
            rotation: {x: 0, y: 45, z: 0},
            scale: {x: 1, y: 1, z: 1}
        }
    ]
}

// Render scene
app.render_scene(scene)

// Update objects functionally
scene.objects[0].rotation <- {
    x: 0,
    y: scene.objects[0].rotation.y + dt * 45,
    z: 0
}
```

### Camera System

```flap
// 2D camera (follows target)
camera_2d := app.camera.create_2d({
    target: {x: player.x, y: player.y},
    offset: {x: 0, y: 0},
    zoom: 1.0,
    rotation: 0,
    bounds: {x: 0, y: 0, w: 2000, h: 2000}
})

// Camera effects
camera_2d <- app.camera.shake(camera_2d, {
    intensity: 10,
    duration: 0.5,
    frequency: 30
})

camera_2d <- app.camera.zoom_to(camera_2d, 2.0, {
    duration: 1.0,
    easing: "ease_in_out"
})

// 3D camera types
camera_fps := app.camera.create_3d({
    type: "fps",
    position: {x: 0, y: 1.8, z: 5},
    yaw: 0,
    pitch: 0,
    fov: 60,
    mouse_sensitivity: 0.1
})

camera_orbit := app.camera.create_3d({
    type: "orbit",
    target: {x: 0, y: 0, z: 0},
    distance: 10,
    min_distance: 2,
    max_distance: 50,
    angle_h: 0,
    angle_v: 30
})

camera_follow := app.camera.create_3d({
    type: "follow",
    target: player,
    offset: {x: 0, y: 2, z: -5},
    smoothness: 0.1
})

// Coordinate conversion
world_pos := app.camera.screen_to_world(camera, screen_x, screen_y)
screen_pos := app.camera.world_to_screen(camera, world_x, world_y, world_z)
```

---

## Physics

### Setup and Configuration

```flap
// Initialize physics world
physics := app.physics.create({
    gravity: {x: 0, y: 9.8},
    iterations: 8,
    timestep: 1.0 / 60.0
})

// Create rigid bodies
body := app.physics.create_body({
    type: "dynamic",        // dynamic, static, kinematic
    position: {x: 0, y: 10},
    rotation: 0,
    linear_damping: 0.1,
    angular_damping: 0.1,
    fixed_rotation: false,
    bullet: false           // Enable CCD for fast objects
})

// Attach colliders
collider := app.physics.attach_collider(body, {
    shape: "box",           // box, circle, polygon, edge
    width: 2,
    height: 2,
    density: 1.0,
    friction: 0.3,
    restitution: 0.5,
    sensor: false,          // Ghost collider (no physics response)
    category: 0x0001,       // Collision filtering
    mask: 0xFFFF
})

// Forces and impulses
app.physics.apply_force(body, {x: 100, y: 0})
app.physics.apply_impulse(body, {x: 0, y: -500})
app.physics.set_velocity(body, {x: 5, y: 0})
app.physics.set_angular_velocity(body, 1.5)

// Queries
bodies := app.physics.query_aabb({x: 0, y: 0, w: 100, h: 100})
bodies := app.physics.query_circle({x: 50, y: 50}, radius: 25)

hit := app.physics.raycast({x: 0, y: 0}, {x: 100, y: 0})
hit {
    -> result {
        printf("Hit at (%v, %v)\n", result.point.x, result.point.y)
        printf("Normal: (%v, %v)\n", result.normal.x, result.normal.y)
        printf("Fraction: %v\n", result.fraction)
    }
}

// Collision callbacks
app.physics.on_collision_begin(player_body, enemy_bodies, (a, b) => {
    printf("Collision started\n")
})

app.physics.on_collision_end(player_body, enemy_bodies, (a, b) => {
    printf("Collision ended\n")
})
```

---

## Input Handling

### Keyboard

```flap
// State checking
app.key_down("space") { player <- jump(player) }
app.key_pressed("escape") { quit() }
app.key_released("f") { toggle_fullscreen() }

// Key state query
is_down := app.input.is_key_down("w")
is_pressed := app.input.is_key_pressed("space")  // True only on frame it was pressed

// Multiple keys
moving := app.key_down("w") or app.key_down("up")

// Key codes
app.input.is_key_down(KEY_LEFT_SHIFT)
```

### Mouse

```flap
// Position
pos := app.mouse.position()
printf("Mouse at (%v, %v)\n", pos.x, pos.y)

delta := app.mouse.delta()  // Movement since last frame

// Buttons (0=left, 1=middle, 2=right)
app.mouse.button_down(0) {
    shoot(pos.x, pos.y)
}

app.mouse.button_pressed(2) {
    open_menu()
}

// Wheel
scroll := app.mouse.wheel()
camera.zoom <- camera.zoom + scroll * 0.1

// Cursor control
app.mouse.set_visible(false)
app.mouse.set_locked(true)  // FPS-style mouse capture
app.mouse.set_cursor("hand")  // default, hand, crosshair, text, etc.
```

### Gamepad

```flap
// Detect connected gamepads
gamepads := app.gamepad.list()
gamepad := gamepads[0]

// Buttons
app.gamepad.button_down(gamepad, "a") { jump() }
app.gamepad.button_pressed(gamepad, "start") { pause() }

// Analog sticks
left_stick := app.gamepad.axis(gamepad, "left_stick")
player.x <- player.x + left_stick.x * speed * dt
player.y <- player.y + left_stick.y * speed * dt

right_stick := app.gamepad.axis(gamepad, "right_stick")
aim_angle := atan2(right_stick.y, right_stick.x)

// Triggers (0.0 to 1.0)
left_trigger := app.gamepad.trigger(gamepad, "left")
right_trigger := app.gamepad.trigger(gamepad, "right")

// Rumble
app.gamepad.rumble(gamepad, {
    low_frequency: 0.5,
    high_frequency: 1.0,
    duration: 0.5
})
```

### Touch (Mobile)

```flap
// Touch events
@ touch in app.touch.active() {
    printf("Touch %v at (%v, %v)\n", touch.id, touch.x, touch.y)
}

app.touch.began(0) {
    -> pos { start_drag(pos.x, pos.y) }
}

app.touch.moved(0) {
    -> pos { update_drag(pos.x, pos.y) }
}

app.touch.ended(0) {
    -> pos { end_drag(pos.x, pos.y) }
}

// Gestures
app.gesture.pinch() {
    -> scale { camera.zoom <- camera.zoom * scale }
}

app.gesture.swipe("left") {
    next_screen()
}
```

### Event System

```flap
// Event loop with pattern matching
@ event in app.events() {
    event.type {
        "quit" -> ret @
        "key_down" -> {
            event.key {
                "escape" -> ret @
                "space" -> jump()
                "f" -> toggle_fullscreen()
            }
        }
        "mouse_down" -> shoot(event.x, event.y)
        "mouse_motion" -> aim(event.x, event.y)
        "window_resize" -> resize(event.width, event.height)
    }
}
```

---

## Audio

### Sound Effects

```flap
// Load audio (automatic format detection: wav, ogg, mp3, flac)
jump_sfx := app.audio.load("jump.wav")
explosion := app.audio.load("explosion.ogg")

// Play sound
app.audio.play(jump_sfx)
app.audio.play(jump_sfx, {
    volume: 0.8,
    pitch: 1.2,
    pan: 0.0          // -1.0 (left) to 1.0 (right)
})

// 3D positional audio
app.audio.play_at(explosion, {
    position: {x: 100, y: 0, z: 50},
    max_distance: 100,
    rolloff: 1.0,
    doppler: 1.0
})

// Control playback
channel := app.audio.play(music)
app.audio.pause(channel)
app.audio.resume(channel)
app.audio.stop(channel)

// Query state
is_playing := app.audio.is_playing(channel)
position := app.audio.get_position(channel)  // Seconds
```

### Music

```flap
// Background music
music := app.audio.load_music("theme.ogg")

app.audio.play_music(music, {
    loop: true,
    volume: 0.7,
    fade_in: 2.0      // Seconds
})

// Transitions
app.audio.fade_out_music(2.0)
app.audio.crossfade_music(old_music, new_music, 1.5)

// Sync with gameplay
beat := app.audio.get_beat(music)  // For rhythm games
```

### Audio Listener (3D Audio)

```flap
// Set listener position (usually camera)
app.audio.set_listener({
    position: {x: camera.x, y: camera.y, z: camera.z},
    forward: {x: 0, y: 0, z: -1},
    up: {x: 0, y: 1, z: 0},
    velocity: {x: 0, y: 0, z: 0}
})
```

---

## Animation & Tweening

### Tween System

```flap
// Basic tween
app.tween(sprite, "x", 500, {
    duration: 2.0,
    easing: "ease_out"
})

// Available easing functions
// linear, ease_in, ease_out, ease_in_out
// ease_in_quad, ease_out_quad, ease_in_out_quad
// ease_in_cubic, ease_out_cubic, ease_in_out_cubic
// ease_in_quart, ease_out_quart, ease_in_out_quart
// ease_in_elastic, ease_out_elastic, ease_in_out_elastic
// ease_in_bounce, ease_out_bounce, ease_in_out_bounce

// Tween multiple properties
app.tween(sprite, {
    x: 500,
    y: 300,
    rotation: 360,
    scale_x: 2.0,
    scale_y: 2.0,
    alpha: 0.0
}, {
    duration: 2.0,
    easing: "ease_in_out",
    delay: 0.5
})

// Callbacks
app.tween(sprite, "x", 500, {
    duration: 2.0,
    on_start: () => { printf("Started\n") },
    on_update: (value) => { printf("Progress: %v\n", value) },
    on_complete: () => { printf("Done\n") }
})

// Sequences
app.tween_sequence([
    {target: sprite, prop: "scale_x", to: 1.5, duration: 0.3},
    {target: sprite, prop: "scale_y", to: 1.5, duration: 0.3},
    {wait: 0.2},
    {target: sprite, prop: "scale_x", to: 1.0, duration: 0.3},
    {target: sprite, prop: "scale_y", to: 1.0, duration: 0.3},
    {callback: () => { on_bounce_complete() }}
])

// Parallel tweens
app.tween_parallel([
    {target: sprite, prop: "x", to: 500, duration: 2.0},
    {target: sprite, prop: "y", to: 300, duration: 2.0},
    {target: sprite, prop: "rotation", to: 360, duration: 2.0}
])

// Control
tween := app.tween(sprite, "x", 500, {duration: 2.0})
app.tween.pause(tween)
app.tween.resume(tween)
app.tween.stop(tween)
app.tween.restart(tween)
```

---

## Tilemap System

### Creation and Loading

```flap
// Load Tiled map format (.tmx, .json)
tilemap := app.tilemap.load("level1.tmx")

// Create programmatically
tilemap := app.tilemap.create({
    width: 20,
    height: 15,
    tile_width: 32,
    tile_height: 32,
    tileset: "tiles.png",
    tiles_per_row: 8
})

// Set tiles
tilemap <- app.tilemap.set_tile(tilemap, {x: 5, y: 10}, tile_id: 42)
tile := app.tilemap.get_tile(tilemap, {x: 5, y: 10})

// Layers
tilemap <- app.tilemap.add_layer(tilemap, "background")
tilemap <- app.tilemap.add_layer(tilemap, "foreground")
tilemap <- app.tilemap.set_layer_visible(tilemap, "background", true)

// Collision
tilemap <- app.tilemap.set_collision_layer(tilemap, "walls")
collisions := app.physics.check_tilemap(player, tilemap, "walls")

// Render
app.draw.tilemap(tilemap)
app.draw.tilemap(tilemap, camera)  // With camera culling
```

---

## UI System

### Basic Widgets

```flap
// Button
button := app.ui.button({
    x: 100, y: 100,
    width: 200, height: 50,
    text: "Click Me",
    font_size: 24,
    color: 0x4CAF50,
    hover_color: 0x45a049,
    on_click: () => { printf("Clicked!\n") }
})

// Label
label := app.ui.label({
    x: 100, y: 50,
    text: "Score: 0",
    font_size: 32,
    color: 0xFFFFFF
})

// Text input
input := app.ui.text_input({
    x: 100, y: 200,
    width: 300, height: 40,
    placeholder: "Enter name...",
    max_length: 20,
    on_change: (text) => { printf("Input: %v\n", text) }
})

// Slider
slider := app.ui.slider({
    x: 100, y: 300,
    width: 200, height: 20,
    min: 0, max: 100,
    value: 50,
    on_change: (value) => { volume <- value }
})

// Checkbox
checkbox := app.ui.checkbox({
    x: 100, y: 400,
    text: "Enable sound",
    checked: true,
    on_change: (checked) => { sound_enabled <- checked }
})
```

### Layout

```flap
// Container
panel := app.ui.panel({
    x: 50, y: 50,
    width: 400, height: 300,
    background: 0x1E1E1E,
    border: {color: 0x333333, width: 2},
    padding: 10
})

// Layout managers
app.ui.layout_vertical(panel, [button1, button2, button3], {
    spacing: 10,
    align: "center"
})

app.ui.layout_horizontal(panel, [label1, label2, label3], {
    spacing: 5,
    align: "left"
})

app.ui.layout_grid(panel, widgets, {
    columns: 3,
    spacing: {x: 10, y: 10}
})
```

---

## Math Utilities

### Vectors

```flap
// 2D vectors
v1 := vec2(10, 20)
v2 := vec2(5, 15)

v3 := v1 + v2              // {x: 15, y: 35}
v3 := v1 - v2              // {x: 5, y: 5}
v3 := v1 * 2               // {x: 20, y: 40}
v3 := v1 / 2               // {x: 5, y: 10}

length := v1.length()
length_sq := v1.length_squared()
normalized := v1.normalize()
distance := v1.distance(v2)
dot := v1.dot(v2)
angle := v1.angle()        // Radians

// 3D vectors
v1 := vec3(1, 2, 3)
v2 := vec3(4, 5, 6)

cross := v1.cross(v2)
```

### Interpolation

```flap
// Linear interpolation
result := lerp(0, 100, 0.5)        // 50
color := lerp_color(0xFF0000, 0x0000FF, 0.5)  // Purple

// Smoothstep
result := smoothstep(0, 100, 0.5)  // Smooth ease

// Clamp
clamped := clamp(150, 0, 100)      // 100
clamped := clamp(-10, 0, 100)      // 0
```

### Angles and Rotation

```flap
// Conversion
rad := deg_to_rad(90)      // π/2
deg := rad_to_deg(3.14159) // 180

// Direction
angle := angle_between(v1, v2)
direction := angle_to_vec2(angle)

// Rotation
v2 := rotate_vec2(v1, angle)
point := rotate_around(point, center, angle)
```

### Random

```flap
// Range
value := random(0, 100)            // Float between 0 and 100
int_value := random_int(1, 6)      // Integer between 1 and 6

// Distributions
value := random_gaussian(mean: 50, stddev: 10)
value := random_exponential(lambda: 1.5)

// Utilities
coin_flip := random_bool()         // 50/50
dice_roll := random_choice([1, 2, 3, 4, 5, 6])
color := random_color()            // Random RGB
```

---

## Asset Management

### Loading

```flap
// Textures
texture := app.load.texture("sprite.png")
texture := app.load.texture("sprite.png", {
    filter: "linear",      // linear, nearest
    wrap: "clamp"          // clamp, repeat, mirror
})

// Audio
sound := app.load.sound("jump.wav")
music := app.load.music("theme.ogg")

// Fonts
font := app.load.font("arial.ttf", size: 24)

// Models
model := app.load.model("player.obj")
model := app.load.model("scene.gltf")

// Shaders
shader := app.load.shader("vertex.glsl", "fragment.glsl")

// Batch loading
assets := app.load.batch({
    "player": "player.png",
    "enemy": "enemy.png",
    "jump": "jump.wav",
    "music": "theme.ogg"
})
```

### Resource Management

```flap
// Unload
app.unload.texture(texture)
app.unload.sound(sound)
app.unload.model(model)

// Unload all
app.unload.all()

// Memory info
mem := app.memory_usage()
printf("Textures: %v MB\n", mem.textures)
printf("Audio: %v MB\n", mem.audio)
printf("Total: %v MB\n", mem.total)
```

---

## Utilities

### Timing

```flap
// Frame timing
dt := app.delta_time()             // Seconds since last frame
fps := app.fps()
time := app.time()                 // Total elapsed seconds

// Timers
timer := app.timer.create(2.0, () => {
    spawn_enemy()
})

app.timer.start(timer)
app.timer.pause(timer)
app.timer.reset(timer)

// Periodic execution
app.every(1.0, () => {
    printf("Once per second\n")
})
```

### Screen and Window

```flap
// Screen dimensions
width := app.screen.width()
height := app.screen.height()
aspect := app.screen.aspect_ratio()

// Window control
app.window.set_title("New Title")
app.window.set_size(1280, 720)
app.window.set_fullscreen(true)
app.window.set_vsync(true)
app.window.center()

// Screenshots
app.window.screenshot("screenshot.png")
```

### Debug

```flap
// Debug drawing
app.debug.draw_point(x, y, color)
app.debug.draw_line(x1, y1, x2, y2, color)
app.debug.draw_rect(x, y, w, h, color)
app.debug.draw_circle(x, y, r, color)
app.debug.draw_text(x, y, "Debug info")

// Physics debug
app.debug.draw_colliders(physics, color)
app.debug.draw_aabbs(physics, color)

// Performance
app.debug.show_fps(x, y)
app.debug.show_memory(x, y)
app.debug.show_stats(x, y)  // Draw calls, entities, etc.
```

---

## Example: Complete 2D Game

```flap
import game

// Game state
State = {
    player: {x: 400, y: 300, vx: 0, vy: 0, health: 100},
    enemies: [],
    bullets: [],
    score: 0,
    game_over: false
}

// Update logic
update := (state, events, dt) => {
    // Input
    dx := 0
    dy := 0
    app.key_down("left") { dx <- -200 }
    app.key_down("right") { dx <- 200 }
    app.key_down("up") { dy <- -200 }
    app.key_down("down") { dy <- 200 }

    // Update player
    player := state.player
    player.x <- clamp(player.x + dx * dt, 0, 800)
    player.y <- clamp(player.y + dy * dt, 0, 600)

    // Shoot
    app.key_pressed("space") {
        bullet := {x: player.x, y: player.y, vy: -500}
        state.bullets <- state.bullets :: bullet
    }

    // Update bullets
    bullets := @ bullet in state.bullets {
        bullet.y <- bullet.y + bullet.vy * dt
        bullet.y > -10 { bullet }  // Keep if on screen
    }

    // Update enemies
    enemies := @ enemy in state.enemies {
        enemy.y <- enemy.y + 50 * dt
        enemy.y < 650 { enemy }    // Remove if off screen
    }

    // Collision detection
    @ bullet in bullets {
        @ enemy in enemies {
            distance := sqrt((bullet.x - enemy.x) ** 2 + (bullet.y - enemy.y) ** 2)
            distance < 30 {
                state.score <- state.score + 100
                // Remove both
            }
        }
    }

    // Spawn enemies
    random() < 0.02 {
        enemy := {x: random(50, 750), y: -50}
        enemies <- enemies :: enemy
    }

    ret {
        player: player,
        bullets: bullets,
        enemies: enemies,
        score: state.score,
        game_over: state.game_over
    }
}

// Render logic
render := (state, app) => {
    app.clear(0x001122)

    // Draw player
    app.draw.circle(state.player.x, state.player.y, 20, 0x00FF00)

    // Draw bullets
    @ bullet in state.bullets {
        app.draw.circle(bullet.x, bullet.y, 5, 0xFFFF00)
    }

    // Draw enemies
    @ enemy in state.enemies {
        app.draw.rect(enemy.x - 20, enemy.y - 20, 40, 40, 0xFF0000)
    }

    // Draw UI
    app.draw.text(10, 10, f"Score: {state.score}", 32, 0xFFFFFF)
    app.draw.text(10, 50, f"Health: {state.player.health}", 24, 0xFFFFFF)

    state.game_over {
        app.draw.text(300, 300, "GAME OVER", 48, 0xFF0000)
    }
}

// Main
main ==> {
    app := game.init("Space Shooter", 800, 600, {vsync: true})
    app.run(State, update, render)
}
```

---

## Example: 3D Scene

```flap
import game

main ==> {
    app := game.init("3D Demo", 800, 600)

    scene := {
        camera: {
            position: {x: 0, y: 5, z: 10},
            look_at: {x: 0, y: 0, z: 0}
        },
        lights: [
            {type: "directional", direction: {x: -1, y: -1, z: -1}},
            {type: "ambient", intensity: 0.2}
        ],
        objects: [
            {
                type: "cube",
                position: {x: 0, y: 0, z: 0},
                rotation: {x: 0, y: 0, z: 0},
                material: {albedo: 0xFF0000}
            }
        ]
    }

    angle := 0

    @ app.running() {
        events := app.poll_events()
        dt := app.delta_time()

        // Rotate cube
        angle <- angle + dt * 45
        scene.objects[0].rotation <- {x: angle, y: angle, z: 0}

        // Render
        app.clear(0x87CEEB)
        app.render_scene(scene)
        app.present()
    }
}
```

---

## Implementation Notes

### SDL3 Mapping

FlapGame is built on SDL3 for cross-platform support:

- Window/rendering: SDL3 + OpenGL/Vulkan
- Input: SDL3 input subsystem
- Audio: SDL3_mixer
- 2D rendering: Hardware-accelerated SDL_Renderer
- 3D rendering: OpenGL 3.3+ / Vulkan backend

### Performance Optimizations

- Automatic sprite batching by texture
- Frustum culling for 3D scenes
- Texture atlasing for sprites
- Spatial hashing for collision detection
- Object pooling for particles
- Draw call minimization

### Platform Support

- Linux: Full support
- macOS: Full support
- Windows: Full support
- Web (WASM): Planned
- iOS/Android: Touch input supported
- Consoles: Possible with SDL3 console backends

---

## Design Comparison

| Feature | FlapGame | Pygame | Raylib | Three.js |
|---------|----------|--------|--------|----------|
| Setup boilerplate | Minimal | Medium | Low | High |
| 2D graphics | First-class | First-class | First-class | Add-on |
| 3D graphics | First-class | None | First-class | First-class |
| Physics built-in | Yes | No | Basic | No |
| Particle system | Yes | Manual | Basic | Manual |
| UI widgets | Yes | Manual | Basic | Manual |
| Audio (3D) | Yes | 2D only | 3D | Requires add-on |
| Animation/tweening | Yes | Manual | No | Requires add-on |
| Tilemap support | Yes | Manual | No | No |
| Camera system | Yes | Manual | Basic | Manual |
| API style | Functional | OOP | Procedural | OOP |
| Learning curve | Low | Medium | Low | High |

---

## Future Enhancements

- **Networking**: Built-in multiplayer support (UDP/WebRTC)
- **Scripting**: Hot-reload Flap code for live development
- **Visual Editor**: Drag-and-drop scene editor
- **Profiler**: Built-in performance analysis tools
- **VR/AR**: OpenXR integration for immersive experiences
- **Advanced Rendering**: PBR materials, shadows, post-processing
- **Asset Pipeline**: Texture compression, model optimization
