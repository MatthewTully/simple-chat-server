package client

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func StartTUI(c *Client) error {
	app := initView(c)
	c.TUI = app
	err := c.TUI.Run()
	if err != nil {
		return err
	}
	return nil
}

func initView(c *Client) *tview.Application {
	app := tview.NewApplication()
	chatLog := createChatLogView().SetChangedFunc(c.textViewChangeHandler)
	textBox := createMsgBoxView()

	textBox.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			input := textBox.GetText()
			c.LastCommand = input
			actionInput(c, input+"\n")
			textBox.SetText("")
		}
	})

	textBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Modifiers() == tcell.ModCtrl {
			if event.Key() == tcell.KeyEnter {
				textBox.SetText(textBox.GetText() + "\n")
				return nil
			}
		}
		if event.Key() == tcell.KeyUp {
			if c.LastCommand != textBox.GetText() {
				textBox.SetText(c.LastCommand)
			}
			return nil
		}
		return event
	})

	chatter_flex := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(chatLog, 0, 1, false).AddItem(textBox, 3, 1, true)
	activeChatters := createActiveChatterView().SetChangedFunc(c.textViewChangeHandler)
	mainView := tview.NewFlex().AddItem(chatter_flex, 0, 5, true).AddItem(activeChatters, 20, 1, false)
	app.SetRoot(mainView, true).EnableMouse(true).EnablePaste(true)

	c.chatView = chatLog
	c.activeUsersView = activeChatters

	return app
}

func (c *Client) textViewChangeHandler() {
	c.TUI.Draw()
}

func createTextView() tview.TextView {
	return *tview.NewTextView().SetDynamicColors(true).SetRegions(true)
}

func createChatLogView() *tview.TextView {
	chatLog := createTextView()
	chatLog.SetTitle("  Chat Log  ") //TODO get from config
	chatLog.SetMaxLines(250)         //TODO get from config //Need to experiment here, see what its like withlimit, without, and if should have scollable or not
	chatLog.SetBorder(true)
	chatLog.SetDynamicColors(true)
	return &chatLog
}

func createActiveChatterView() *tview.TextView {
	usrList := createTextView()
	usrList.SetTitle("  Active Users  ")
	usrList.SetBorder(true)
	return &usrList
}

func createMsgBoxView() *tview.InputField {
	txtBox := tview.NewInputField()
	txtBox.SetPlaceholder("Enter message here...")
	txtBox.SetBorder(true)
	txtBox.SetFieldBackgroundColor(tcell.ColorDefault)
	txtBox.SetFieldTextColor(tcell.ColorDefault)
	txtBox.SetPlaceholderTextColor(tcell.ColorDefault)

	txtBox.SetBorderPadding(0, 0, 1, 1)

	return txtBox
}
