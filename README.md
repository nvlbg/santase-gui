Santase
=======

This is a GUI for the game of santase (also known as
[sixty-six](https://en.wikipedia.org/wiki/Sixty-Six_(card_game))).

![Preview](https://raw.githubusercontent.com/nvlbg/santase-gui/master/assets/preview.png)

This project uses [santase-ai](https://github.com/nvlbg/santase-ai/) as an
underlying library for choosing the moves of the opponent. You can create your
own agent with it and use this GUI to test how it performs.

Installation
------------
This project uses go modules introduced in Go 1.11 for dependency resolution.
Having Go 1.11 installed running this project is as simple as:

```bash
# clone this repository (outside of $GOPATH)
cd santase-gui
go run .
```

Development
-----------
Here are some tips if you want to hack with this project.

### Debug mode
In the UI press `F12` to enter debug mode. In it you can see the hidden cards
and other information that can be useful when developping.

### Replace santase-ai dependency to a local copy
You may need to edit something in the santase-ai library. To make this easier
edit `go.mod` file and add the following line:

```
replace github.com/nvlbg/santase-ai => /path/to/local/santase-ai
```

This way the local copy of santase-ai will be used when running the project.

### Use different AI agent
By default the GUI will use the ISMCTS agent that comes with santase-ai for
choosing the moves for one player and the user for choosing the moves for the
other player. You can change the opponent agent in the `main` function to see
how a different agent would play. Another possibility is to play two different
AI agents against each other. All you need to do is initialize the other agent
and pass it as a second argument to `NewGame` in the `main` function.

### Replaying a game
By default every time the project runs it generates a different game. Sometimes
it may be useful to play the same game (same card deal) again, for example if
you work on an AI and you want to see how different methods would play out.
To do so, you can change how the RNG is seeded in the NewGame function.

License
-------
This project is licensed under the MIT License.
