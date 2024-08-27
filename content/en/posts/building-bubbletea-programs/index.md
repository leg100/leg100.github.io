---
title: Building Bubbletea Programs
slug: building-bubbletea-programs
date: 2024-08-24T10:26:59+01:00
tags:
  - bubbletea
  - go
# draft: true
---

## 1. Keep the event loop fast {#keepfast}

Bubbletea processes messages in an event loop:

```go
func (p *Program) eventLoop(model Model, cmds chan Cmd) (Model, error) {
    for {
        select {
        case msg := <-p.msgs:
            // handle quit, window resize, etc
            // ...
            var cmd Cmd
            model, cmd = model.Update(msg) // run update
            cmds <- cmd                    // process command (if any)
            p.renderer.write(model.View()) // send view to renderer
        }
    }
}
```

An event is received from the commands channel and sent to the `Update()` method on your model. The returned command is sent to an unbuffered channel, which a receiver invokes in a go routine. Your model's `View()` method is then invoked before repeating the loop and processing the next event.

Therefore Bubbletea can only process events as fast as as your `Update()` and `View()` methods. You want these methods to be fast otherwise your program may experience lag, resulting in an unresponsive UI. If your program generates a lot of messages they can back up and the program may appear to stall: a user presses a key and nothing happens for an indetermine amount of time.

The key to writing a fast model is to offload expensive operations to a `tea.Cmd`:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
            // don't do this:
            // time.Sleep(time.Minute)

            // do this:
            return func() tea.Msg {
                time.Sleep(time.Minute)
            }
	}
	return m, nil
}
```

## 2. Dump messages to a file

When debugging it can be invaluable to see the messages your model is receiving. To do so, dump every message to a file and tail that file in another terminal. I recommend [spew](https://github.com/davecgh/go-spew) to pretty print the messages. The following dumps messages when `DEBUG` is set:

```go
type model struct {
	dump io.Writer
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.dump != nil {
		spew.Fdump(m.dump, msg)
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string { return "" }

func main() {
	var dump *os.File
	if _, ok := os.LookupEnv("DEBUG"); ok {
		var err error
		dump, err = os.OpenFile("messages.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			os.Exit(1)
		}
	}
	p := tea.NewProgram(model{dump: dump})

	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
```

Run the program then tail `messages.log` in another terminal:

```
> tail -f messages.log 
(tea.KeyMsg)  
(tea.KeyMsg) m
(tea.KeyMsg) e
(tea.KeyMsg) s
(tea.KeyMsg) s
(tea.KeyMsg) a
(tea.KeyMsg) g
(tea.KeyMsg) e
(tea.KeyMsg) s
(tea.KeyMsg) enter
(tea.KeyMsg) ctrl+c
```

## 3. Live reload code changes

Web app developers use [livereload](https://github.com/livereload/livereload-js) to see the effects of code changes in near real-time in the browser. You should do the same for your TUI. I've cobbled together a couple of scripts for PUG:

```bash
#!/usr/bin/env bash

# in foreground, continously run app
while true; do
    _build/pug -w ./demo/ -d -l debug
done
```

```bash
#!/usr/bin/env bash

# watch code changes, trigger re-build, and kill process 
while true; do
    go build -o _build/pug && pkill -f '_build/pug'
    inotifywait -e attrib $(find . -name '*.go') || exit
done
```

I run the scripts in separate terminals. Whenever I save code changes the changes are visible pretty much immediately afterwards thanks to Go's fast compilation. This solution could be improved to handle a couple of shortcomings:

* If I've messed up and PUG panics upon startup then it'll go into a death spin, restarting umpteen times a second until the error is fixed. A backoff mechanism would help here.
* Starting and stopping these scripts is a bit of a pain. To stop them, I first need to stop the rebuild script, then use `kill` to terminate the other script.

There are several tools out there like [air](https://github.com/air-verse/air) that do "live reload for the CLI". But [I found they didn't work](https://github.com/charmbracelet/bubbletea/issues/150) with programs that are expecting standard input to be a TTY, which includes TUI's, but users in that same issue [reported success with watchexec](https://github.com/charmbracelet/bubbletea/issues/150#issuecomment-988857894).

## 4. Use receiver methods on your model judiciously

In Go, a method receiver can be passed as either a value or a pointer. When in doubt, one typically uses a pointer receiver, with a value receiver reserved for [particular use cases](https://google.github.io/styleguide/go/decisions#receiver-type).

It can throw Go programmers then that the documented Bubbletea models all have *value* receivers. It may be due to the fact Bubbletea is based on the [Elm Architecture](https://guide.elm-lang.org/architecture/), which is a purely functional pattern, where functions cannot change their internal state, and in Go a method with a value receiver cannot modify its receiver.

However, you are free to set whatever receiver type you like. If you use a pointer receiver for your model and you make, say, a change to the model in `Init()` then that change is persisted:

```go
type model struct {
	content string
}

func (m *model) Init() tea.Cmd {
	m.content = "initialized\n"
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyCtrlC:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) View() string { return m.content }

func main() {
	p := tea.NewProgram(&model{content: "uninitalized"})
	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
```

Returns:

```
initialized
```

However, bear in mind that Bubbletea only renders changes in response to messages: 

```go
type model struct {
	content string
}

func (m *model) Init() tea.Cmd {
	go func() {
		<-time.After(time.Second)
		m.content = "initialized\n"
	}()
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) View() string { return m.content }

func main() {
	p := tea.NewProgram(&model{content: "uninitalized"})
	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
```

Returns:

```
uninitialized
```

...but only once a key is pressed several seconds later:

```
initialized
```

For this reason, making changes to the model is best done in the normal message flow, either in `Update()` or via a `tea.Cmd`. However, using a pointer receiver is useful on helper methods:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.updateDimensions(msg.Width, msg.Height)
	}
	return m, nil
}

func (m *model) updateDimensions(width, height int) {
	m.width = width
	m.height = height
}
```

There is a [Github discussion](https://github.com/charmbracelet/bubbletea/discussions/434) on this subject, where there are different opinions on when to use pointer receivers.

## 5. Events are not necessarily received in the order they are sent

In Go, if you have more than one go routine sending to a channel, the order in which the sends and receives occur is unspecified:

https://go.dev/play/p/G4-o8F6PsvU

```go
func main() {
	ch := make(chan int)
	wg := sync.WaitGroup{}

	for i := range 10 {
		wg.Add(1)
		go func() {
			ch <- i
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for i := range ch {
		fmt.Printf("%d\n", i)
	}
}
````

Running the above can return something like this:

```
0512347689
```

Now, in Bubbletea, events arrive from a number of sources, including:

1) From user input, key presses, mouse movements, etc.
2) Messages from tea commands (`tea.Cmd`).
3) Explicitly sent using `Send(msg)`.
4) Signals such as window resize, suspend etc.

User input events are sent in a single routine:

```go
// readAnsiInputs reads keypress and mouse inputs from a TTY and produces messages
// containing information about the key or mouse events accordingly.
func readAnsiInputs(ctx context.Context, msgs chan<- Msg, input io.Reader) error {
    // ...
    for {
        // Read and block.
        numBytes, err := input.Read(buf[:])
        // ...
        select {
        case msgs <- msg:
        // ...
```

User input events *are* therefore sent in order. Just as well otherwise entering words into, say, a text input would end up as gibberish.

However, Bubbletea commands are executed concurrently in separate go routines: 

```go
// handleCommands runs commands in a goroutine and sends the result to the
// program's message channel.
func (p *Program) handleCommands(cmds chan Cmd) chan struct{} {
    // ...
    case cmd := <-cmds:
        if cmd == nil {
            continue
        }
        // ...
        go func() {
            msg := cmd() // this can be long.
            p.Send(msg)
        }()
```

As such, the messages they produce are sent in an unspecified order. Even if a command finishes before another command there is no guarantee its message will be sent and received first.

The following demonstrates the point by explicitly sending messages using `Send(msg)` concurrently:

```go
type model struct {
	received []int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case nMsg:
		m.received = append(m.received, int(msg))
	case tea.KeyMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	return fmt.Sprintf("%v", m.received)
}

type nMsg int

func main() {
	p := tea.NewProgram(model{})

	for i := range 10 {
		go p.Send(nMsg(i))
	}

	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
```

Running the program produces:

```
> go run main.go
[0 1 9 8 5 6 4 2 3 7]
```

You cannot rely on the ordering of concurrently sent messages. If ordering matters you have a number of workarounds:

1. Update model directly within `Update()`:

    ```go
    func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
        switch msg.(type) {
        case tea.KeyMsg:
            m.ordered = append(m.ordered, nextItem())
            // return func() tea.Msg {
            //   return nextItem()
            // }
        }
        return m, nil
    }
    ```

    This may contradict with [keeping Update fast]({{< ref "#keepfast" >}}).

2. Use `tea.Sequence` to run commands in sequence, one at a time:

    ```go
    func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
        switch msg.(type) {
        case tea.KeyMsg:
            return m, tea.Sequence(doThis, thenThat)
        }
        return m, nil
    }
    ```

    See the [docs](https://pkg.go.dev/github.com/charmbracelet/bubbletea#Sequence).

And finally, if ordering matters then try to refactor your program so that it doesn't. Embrace the chaos of concurrency.

## 6. Build a tree of models

Any non-trivial Bubbletea program outgrows a single model. There's a good chance you're using Charm's [bubbles](https://github.com/charmbracelet/bubbles), which are models in their own right, each with a `Init()`, `Update()`, and `View()`. You embed these models within your own model. The same applies to your own code: you may want to push your own components into separate models. The original model then becomes the "top-level" model, whose role becomes merely a message router and screen compositor, responsible for routing messages to the correct "child" models, and populating a layout with content from the child models' `View()` methods.

And in turn the child models may embed models too, forming a tree of models: the root model receives all messages, which are relayed down the tree to the relevant child model, and the resulting model and command(s) are passed back up the tree, to be returned by the root model's `Update()` method. The same traversal then occurs with the rendering: the root model's `View()` method is called, which in turn calls child models' `View()` methods, and the resulting strings are passed back up the tree to be joined together and returned to the renderer.

This is the model tree implemented in PUG, with arrows illustrating the routing of messages from model to model:

![tree of models in PUG](./model-tree.svg)

Note: the diagram gives the false impression that some models are shared, but in actual fact there is a separate instance of each model, e.g. there is a separate instance of `split` for each of `tasks`, `task groups`, and `state`.

## 7. Layout arithmetic is error-prone

You are responsible for ensuring your program fits in the terminal. Its dimensions arrive in a `tea.WindowSizeMsg` message, which is sent shortly after the program starts, and whenever the terminal is resized. Your model records the dimensions and uses them when rendering to calculate the sizes of widgets.

In this app, there are three widgets: a header, content, and a footer:

```go
type model struct {
	width, height int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m model) View() string {
	header := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Render("header")
	footer := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Render("footer")

	content := lipgloss.NewStyle().
		Width(m.width).
        // accomodate header and footer
		Height(m.height-1-1).
		Align(lipgloss.Center, lipgloss.Center).
		Render("content")

	return lipgloss.JoinVertical(lipgloss.Top, header, content, footer)
}

func main() {
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
```

Which produces:

![working layout](./layout-arithmetic-bad-but-works/screenshot.png)

The header and footer are of fixed sizes, and the content widget takes whatever space is leftover.

The code is then amended to add a border to the bottom of the header:

```go
func (m model) View() string {
	header := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		Render("header")
	footer := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Render("footer")

	content := lipgloss.NewStyle().
		Width(m.width).
        // accomodate header and footer
		Height(m.height-1-1).
		Align(lipgloss.Center, lipgloss.Center).
		Render("content")

	return lipgloss.JoinVertical(lipgloss.Top, header, content, footer)
}
```

But this breaks the layout, forcing the header off the terminal:

![broken layout](./layout-arithmetic-broken/screenshot.png)

The problem is that the arithmetic has not been updated to accomodate the border. The code is brittle, using hard coded heights which can easily be forgotten about when updating code. The fix is to use lipgloss's `Height()` and `Width()` methods to reference heights and widths of widgets:

```go
func (m model) View() string {
	header := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		Render("header")
	footer := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Render("footer")

	content := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-lipgloss.Height(header)-lipgloss.Height(footer)).
		Align(lipgloss.Center, lipgloss.Center).
		Render("content")

	return lipgloss.JoinVertical(lipgloss.Top, header, content, footer)
}
```

Which fixes the layout:

![fixed layout](./layout-arithmetic-good/screenshot.png)

Now when changes are made to widget sizes the layout adapts accordingly.

As your program gets more complex, with more widgets and more models, it's important to be disciplined with setting dimensions, to avoid frustratingly trying to track down what has caused the layout to break.

## 8. Recovering your terminal

Bubbletea recovers gracefully from panics occuring in the event loop but not if it occurs in a command:

```go
type model struct{}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, func() tea.Msg {
		panic("error")
	}
}

func (m model) View() string { return "" }

func main() {
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
```

Which when invoked causes this:

![unreset terminal following panic](./terminal-not-reset/panic.gif)

Note the stack trace is malformed and there is no cursor. Because the panic was not recovered, the terminal was not reset to its previous settings, with raw mode disabled etc, and any characters you type are no longer echoed.

The recover your terminal, reset it:

```bash
reset
```

You should then regain your cursor and your terminal's previous settings.

Note: there is an [open issue](https://github.com/charmbracelet/bubbletea/issues/234) for this bug.

## 9. Use teatest for end-to-end tests

For end to end testing of your TUI, Charm have developed [teatest](https://github.com/charmbracelet/x/tree/main/exp/teatest), which they introduced last year in a [blog article](https://charm.sh/blog/teatest/).

Here's an program that runs and then quits upon confirmation from the user:

```go
type model struct {
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, nil
		}
		if m.quitting {
			switch {
			case msg.String() == "y":
				return m, tea.Quit
			default:
				m.quitting = false
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Quit? (y/N)"
	} else {
		return "Running."
	}
}
```

And here's the test:

```go
func TestQuit(t *testing.T) {
	m := model{}
	tm := teatest.NewTestModel(t, m)

	waitForString(t, tm, "Running.")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	waitForString(t, tm, "Quit? (y/N)")

	tm.Type("y")

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func waitForString(t *testing.T, tm *teatest.TestModel, s string) {
	teatest.WaitFor(
		t,
		tm.Output(),
		func(b []byte) bool {
			return strings.Contains(string(b), s)
		},
		teatest.WithCheckInterval(time.Millisecond*100),
		teatest.WithDuration(time.Second*10),
	)
}
```

As you can see the test emulates the user pressing keys and checking that the program responds in kind, before checking the program has finished.

While this particular test is only checking for sub-strings, the blog article linked above shows how teatest supports using "golden files", where the entire output is captured the first time the test is run, and subsequent tests then check the content matches the captured output. That's useful for regression testing of content, but does mean you need to re-generate the golden files everytime you make even minor changes to the content of your program.

## 10. Record demos and screenshots on VHS

This one isn't about building your program per se but testing and documenting and presenting it to your audience. Charm make lots of nice tools for the terminal and [VHS](https://github.com/charmbracelet/vhs) is one such tool that meets this requirement to a tee.

With one declarative script you can produce both animated gifs and screenshot pngs. Here is a snippet from the script for PUG:

```
Output demo/demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Framerate 24
Set Padding 5

Hide
Type `go run main.go` Enter
Sleep 1s
Show

# init all modules
Ctrl+a Sleep 0.5s Type "i"
# we're taken to the init task group page
Sleep 0.5s
# preview output for several tasks
Down Sleep 0.5s Down Sleep 0.5s Down Sleep 0.5s

...
# go back to modules
Type "m" Sleep 0.5s
# take screen shot of modules (sleep to ensure page doesn't switch too soon)
Screenshot demo/modules.png Sleep 0.5s
```

VHS calls the script a "tape" and you record the resulting gif by running:

```
vhs demo.tape
```

Which then outputs the animated gif specified in the tape. Here is the full animated gif for PUG:

![pug demo](https://github.com/leg100/pug/raw/master/demo/demo.gif)

Commit the tape alongside your code. You can opt to record a new video as part of your build pipeline, and then watch the video prior to making a release. The same tape can produce screenshots which again form part of your testing as well documentation.

## 11. And more...

I'll endeavour to keep adding more "best practices" as I come across them. But there is no substitute for reading the code of Bubbletea and Bubbletea-based projects. I invite you to read the code of [PUG](https://github.com/leg100/pug), which implements several components that may be of use to your own project:

* Table widget, with selections, sorting, filtering, and custom row rendering.
* Split model: split screen with table and preview panes; adjustable/toggleable split.
* Navigator: makes and caches models, history tracker.
* Integration tests: using teatest for end-to-end testing.
