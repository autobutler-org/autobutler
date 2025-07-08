# UI

## Important Links

- [Templ](https://templ.guide/)
- [HTMX](https://htmx.org/)
  - [Essays](https://htmx.org/essays/)
  - [Reference](https://htmx.org/reference/)
- [MDN docs](https://developer.mozilla.org/en-US/)

## Overview

The UI of Autobutler is built using Templ, a templating language that allows for dynamic
HTML generation in Golang. The UI is structured around components, which can be reused
and composed to create complex interfaces.

## FAQ

### Why are some components public and some private?

First off, when we say "public" we mean that the component is available in Golang's import
system.

Given that, private components can be understood as components used in the internals of
the public components.

For example, the [`chat`](../../ui/components/chat/component.templ) component is public, but is composed of several private components
such as [`input`](../../ui/components/chat/input.templ) and [`spinner`](../../ui/components/chat/spinner.templ).

However, the `chat` interface does utilize the [`message`](../../ui/components/chat/message/component.templ)
component as a public component. The `message` component is used directly by the server router
in order to return error messages to the main interface.

### How do I add a new component?

For a public component, make a new folder in [`components`](../../ui/components) for the component. If the component
makes sense to be semantically nested in another public component [such as `chat/message`](../../ui/components/chat/messsage/component.templ),
go ahead and do such a thing. A public component exposes a single function as it's interface `Component(args...)`

Take this simple example of an early version of the `chat` component:

```templ
// Notice the package name represents the public component
package chat

import (
    "autobutler/internal/llm"
    // This chat component imports another public component for composition
    "autobutler/internal/server/ui/components/chat/message"
)

const spinnerId = "spinner"

templ Component(messages []llm.ChatMessage) {
    <div class="h-screen flex flex-col">
        // This is a private component of the public chat component, a header only for this page
        @header()
        <div class="flex-1 overflow-y-auto p-4">
            <div id="messages" class="max-w-4xl mx-auto space-y-4">
                for _, msg := range messages {
                    @message.Component(msg)
                }
            </div>
            @spinner(spinnerId)
        </div>
        @input()
    </div>
}
```

### How do I use a component I just made?

Using a component directly from a `view` is as follows:

```templ
package views

import (
	"time"
	"autobutler/internal/llm"
	// We import the public components here
	"autobutler/internal/server/ui/components/chat"
	"autobutler/internal/server/ui/components/header"
	"autobutler/internal/server/ui/components/footer"
	"autobutler/internal/server/ui/components/body"
)

templ Chat() {
	<!DOCTYPE html>
	<html lang="en">
		// Notice how the components are referenced like so: `name`.Component()
		@header.Component()
		@body.Component() {
			@chat.Component(
				[]llm.ChatMessage{
					{Role: llm.ChatRoleSystem, Content: "Welcome to the Autobutler!", Timestamp: llm.GetTimestamp(time.Now())},
				},
			)
		}
		@footer.Component()
	</html>
}
```

To use the component directly in a server route, you can do this directly
with the HTTP writer provided by [`gin`](https://gin-gonic.com/).

Example API route that renders a `chat.message`:

```go
apiRoute(apiV1Group, "GET", "/ai-chat", func(c *gin.Context) {
	// Checks if the requester wants HTML, and if not, defaults to JSON
	isHtml := c.GetHeader("Accept") == "text/html"
	// Grabs a query param by key name
	prompt := c.Query("prompt")
	response, err := llm.RemoteLLMRequest(prompt)
	if err != nil {
		if isHtml {
			// Calling `message.Component` returns an object representing the component to be rendered,
			// embedding whatever arguments are pased into it
			messageComponent := message.Component(llm.ErrorChatMessage(err))
			// The component can now be rendered (even repeatedly), as long as it is provided
			// the request context and the Writer interface from the HTTP framework
			if err := messageComponent.Render(c.Request.Context(), c.Writer); err != nil {
				c.Status(500)
				return
			}
		} else {
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
		}
		return
	}
	if isHtml {
		// Same as before for the error, but now returning a good response
		messageComponent := message.Component(llm.FromCompletionToChatMessage(*response))
		if err := messageComponent.Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(500)
			return
		}
	} else {
		c.JSON(200, response)
	}
})
```

### What is a [`view`](../../ui/views/)?

A `view` is equivalent to a page, and it should b composed entirely of public components.

Check out the [`chat` view](../../ui/views/chat.templ) as a complete example.
