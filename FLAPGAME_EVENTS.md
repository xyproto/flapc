# FLAPGAME Event System Design

## Philosophy

Games need **two types of input handling**:

1. **Continuous State**: Keys held down (movement)
2. **Discrete Events**: Key presses, mouse clicks (actions)

Flap's event system supports both naturally.

---

## Continuous State Checking

For smooth movement, check key states every frame:

```flap
@ app.running() {
    dt := app.delta_time()

    // Reset velocity
    player.vx <- 0.0
    player.vy <- 0.0

    // Check held keys (continuous state)
    app.key_down("w") { player.vy <- -player.speed }
    app.key_down("s") { player.vy <- player.speed }
    app.key_down("a") { player.vx <- -player.speed }
    app.key_down("d") { player.vx <- player.speed }

    // Update position
    player.x <- player.x + player.vx * dt
    player.y <- player.y + player.vy * dt
}
```

---

## Event-Based Actions

For one-time actions, use event pattern matching:

```flap
@ app.running() {
    dt := app.delta_time()

    // Process discrete events
    @ event in app.events() {
        event.type {
            "key_pressed" -> {
                event.key {
                    "space" -> { shoot() }
                    "r" -> { reload() }
                    "escape" -> { pause() }
                }
            }
            "mouse_click" -> {
                shoot_at(event.x, event.y)
            }
            "window_close" -> { ret @ }
        }
    }
}
```

---

## Complete Example

```flap
import game

main ==> {
    app := game.init("Space Shooter", 800, 600, {vsync: true})

    // Game state
    state := {
        player: {
            x: 400.0, y: 500.0,
            vx: 0.0, vy: 0.0,
            speed: 200.0,
            health: 100
        },
        enemies: [],
        bullets: [],
        score: 0,
        paused: 0
    }

    // Game loop
    @ app.running() {
        dt := app.delta_time()

        // ===== CONTINUOUS INPUT (Movement) =====
        state.player.vx <- 0.0
        state.player.vy <- 0.0

        state.paused == 0 {
            -> {
                app.key_down("w") { state.player.vy <- -state.player.speed }
                app.key_down("s") { state.player.vy <- state.player.speed }
                app.key_down("a") { state.player.vx <- -state.player.speed }
                app.key_down("d") { state.player.vx <- state.player.speed }
            }
        }

        // ===== DISCRETE EVENTS (Actions) =====
        @ event in app.events() {
            event.type {
                "key_pressed" -> {
                    event.key {
                        "space" -> {
                            state.paused == 0 {
                                -> {
                                    bullet := {
                                        x: state.player.x,
                                        y: state.player.y,
                                        vy: -500.0
                                    }
                                    state.bullets <- bullet :: state.bullets
                                }
                            }
                        }
                        "escape" -> {
                            state.paused <- state.paused == 0 { 1 ~> 0 }
                        }
                        "r" -> {
                            // Reset game
                            state.player.health <- 100
                            state.score <- 0
                            state.enemies <- []
                            state.bullets <- []
                        }
                    }
                }
                "mouse_click" -> {
                    // Shoot towards mouse position
                    dx := event.x - state.player.x
                    dy := event.y - state.player.y
                    len := ((dx ** 2) + (dy ** 2)) ** 0.5

                    bullet := {
                        x: state.player.x,
                        y: state.player.y,
                        vx: (dx / len) * 500.0,
                        vy: (dy / len) * 500.0
                    }
                    state.bullets <- bullet :: state.bullets
                }
                "window_close" -> { ret @ }
                "window_resize" -> {
                    app.resize(event.width, event.height)
                }
            }
        }

        // ===== UPDATE =====
        state.paused == 0 {
            -> {
                // Update player
                state.player.x <- state.player.x + state.player.vx * dt
                state.player.y <- state.player.y + state.player.vy * dt

                // Clamp to screen
                state.player.x <- clamp(state.player.x, 0.0, 800.0)
                state.player.y <- clamp(state.player.y, 0.0, 600.0)

                // Update bullets
                updated_bullets := []
                @ bullet in state.bullets max 10000 {
                    bullet.x <- bullet.x + bullet.vx * dt
                    bullet.y <- bullet.y + bullet.vy * dt

                    // Keep if on screen
                    bullet.y > -10 and bullet.y < 610 {
                        -> { updated_bullets <- bullet :: updated_bullets }
                    }
                }
                state.bullets <- updated_bullets

                // Update enemies
                @ enemy in state.enemies max 10000 {
                    enemy.y <- enemy.y + 50.0 * dt
                }

                // Spawn enemies
                random() < 0.02 {
                    -> {
                        enemy := {x: random(50.0, 750.0), y: -50.0}
                        state.enemies <- enemy :: state.enemies
                    }
                }
            }
        }

        // ===== RENDER =====
        app.clear(0x001122)

        // Draw player
        app.draw.circle(
            state.player.x,
            state.player.y,
            20.0,
            0x00FF00
        )

        // Draw bullets
        @ bullet in state.bullets max 10000 {
            app.draw.circle(bullet.x, bullet.y, 5.0, 0xFFFF00)
        }

        // Draw enemies
        @ enemy in state.enemies max 10000 {
            app.draw.rect(enemy.x - 20.0, enemy.y - 20.0, 40.0, 40.0, 0xFF0000)
        }

        // Draw UI
        app.draw.text(10.0, 10.0, f"Score: {state.score}", 32, 0xFFFFFF)
        app.draw.text(10.0, 50.0, f"Health: {state.player.health}", 24, 0xFFFFFF)

        state.paused {
            -> {
                app.draw.text(300.0, 300.0, "PAUSED", 48, 0xFFFF00)
            }
        }

        app.present()
    }
}
```

---

## Event Types Reference

### Keyboard Events

```flap
event.type {
    "key_down" -> { }      // Key is currently held
    "key_pressed" -> { }   // Key was just pressed (one-time)
    "key_released" -> { }  // Key was just released
}

event.key {
    "w" "a" "s" "d" -> { }           // Movement keys
    "space" "enter" "escape" -> { }  // Action keys
    "shift" "ctrl" "alt" -> { }      // Modifiers
    "f1" "f2" ... "f12" -> { }       // Function keys
}
```

### Mouse Events

```flap
event.type {
    "mouse_move" -> {
        x := event.x
        y := event.y
    }
    "mouse_click" -> {
        button := event.button  // 0=left, 1=middle, 2=right
        x := event.x
        y := event.y
    }
    "mouse_wheel" -> {
        delta := event.delta
    }
}
```

### Window Events

```flap
event.type {
    "window_close" -> { ret @ }
    "window_resize" -> {
        app.resize(event.width, event.height)
    }
    "window_focus" -> { pause <- 0 }
    "window_blur" -> { pause <- 1 }
}
```

### Gamepad Events

```flap
event.type {
    "gamepad_button" -> {
        event.button {
            "a" -> { jump() }
            "b" -> { attack() }
            "start" -> { pause() }
        }
    }
    "gamepad_axis" -> {
        // event.axis = "left_stick_x", "left_stick_y", etc.
        // event.value = -1.0 to 1.0
    }
}
```

---

## Advanced Patterns

### State Machine with Events

```flap
state := { mode: "menu" }

@ event in app.events() {
    state.mode {
        "menu" -> {
            event.key {
                "enter" -> { state.mode <- "playing" }
                "escape" -> { ret @ }
            }
        }
        "playing" -> {
            event.key {
                "escape" -> { state.mode <- "paused" }
                "space" -> { shoot() }
            }
        }
        "paused" -> {
            event.key {
                "escape" -> { state.mode <- "playing" }
                "r" -> { state.mode <- "menu" }
            }
        }
    }
}
```

### Event Filtering

```flap
@ event in app.events() {
    // Only process if not paused
    paused == 0 {
        -> {
            event.type {
                "key_pressed" -> { handle_key(event.key) }
            }
        }
    }

    // Always process these
    event.type {
        "window_close" -> { ret @ }
        "key_pressed" -> {
            event.key {
                "escape" -> { paused <- not paused }
            }
        }
    }
}
```

### Combo Detection

```flap
input_buffer := []
last_input_time := 0.0

@ event in app.events() {
    event.type {
        "key_pressed" -> {
            current_time := app.time()

            // Reset buffer if too much time passed
            (current_time - last_input_time) > 1.0 {
                -> { input_buffer <- [] }
            }

            // Add to buffer
            input_buffer <- input_buffer + [event.key]
            last_input_time <- current_time

            // Check for combos
            input_buffer == ["up", "up", "down", "down"] {
                -> {
                    activate_konami_code()
                    input_buffer <- []
                }
            }
        }
    }
}
```

---

## Summary

**Use continuous state for**:
- Player movement (WASD)
- Camera control
- Anything that should respond while key is held

**Use events for**:
- Shooting/attacking (space, mouse click)
- Menu navigation (enter, escape)
- One-time actions (reload, pause)
- Window events

This hybrid approach gives you the best of both worlds:
- Smooth, responsive movement
- Clean event handling
- Pattern matching power
- Functional data flow
