package auth

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/chromedp/cdproto/storage"
	chrome "github.com/chromedp/chromedp"
)

var (
	execPath     chrome.ExecAllocatorOption
	headless     chrome.ExecAllocatorOption
	loginTimeout = 5 * time.Minute
)

// SetExecPath sets the path to the Chromium or Google Chrome binary.
func SetExecPath(path string) {
	execPath = chrome.ExecPath(path)
}

func SetHeadless(startHeadless bool) {
	headless = chrome.Flag("headless", startHeadless)
}

func SetLoginTimeout(timeout time.Duration) {
	loginTimeout = timeout
}

type AuthInfo struct {
	SessionID string
	XSRFToken string
}

func wait(dur time.Duration) chrome.ActionFunc {
	return chrome.ActionFunc(func(context.Context) error {
		time.Sleep(dur)

		return nil
	})
}

const (
	typingDelay  = 25 * time.Millisecond
	typingJitter = 30 // also ms
)

func simulateTyping(sel interface{}, text string, verbose bool, actionName string) []chrome.Action {
	actions := make([]chrome.Action, 0, 2*len(text)+1)
	if verbose {
		actions = append(actions, &actionLogger{
			log:    true,
			name:   actionName,
			Action: chrome.ActionFunc(func(ctx context.Context) error { return nil }),
		})
	}

	for _, r := range text {
		jitter := time.Duration(rand.Intn(typingJitter)) * time.Millisecond
		actions = append(actions, chrome.SendKeys(sel, string(r)), wait(typingDelay+jitter))
	}

	return actions
}

type actionLogger struct {
	log  bool
	name string
	chrome.Action
}

func (a *actionLogger) Do(ctx context.Context) error {
	if a.log {
		log.Println("<login>", a.name)
	}
	return a.Action.Do(ctx)
}

const loginAnimationTimeout = 5 * time.Second

func Login(username, password string, verbose bool) (*AuthInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), loginTimeout)
	defer cancel()

	opts := append(chrome.DefaultExecAllocatorOptions[:],
		chrome.DisableGPU,
	)
	if execPath != nil {
		opts = append(opts, execPath)
	}
	if headless != nil {
		opts = append(opts, headless)
	}
	allocCtx, aCancel := chrome.NewExecAllocator(ctx, opts...)
	defer aCancel()

	taskCtx, tCancel := chrome.NewContext(allocCtx, chrome.WithLogf(log.Printf))
	defer tCancel()

	withLog := func(name string, action chrome.Action) chrome.Action {
		return &actionLogger{log: verbose, name: name, Action: action}
	}

	info := AuthInfo{}
	actions := []chrome.Action{
		withLog("navigate to cloud.cambiumnetworks.com",
			chrome.Navigate("https://cloud.cambiumnetworks.com/")),
		withLog("waiting for page to load",
			chrome.WaitVisible(`form.signin`)),
		withLog("navigate to SSO login",
			chrome.Click(`form.signin a.btn-primary`, chrome.NodeVisible)),
		withLog("waiting for page to load",
			chrome.WaitVisible(`input[name="email"`)),
	}
	actions = append(actions, simulateTyping(`input[name="email"`, username, verbose, "entering email")...)
	actions = append(actions,
		withLog("navigate to next page",
			chrome.Click(`button[name="next"]`)),
		withLog("waiting for page to load",
			chrome.WaitVisible(`input[name="password"]`)),
	)
	actions = append(actions, simulateTyping(`input[name="password"]`, password, verbose, "entering password")...)
	actions = append(actions,
		withLog("ticking 'remember me' checkbox",
			chrome.Click(`input[name="remember"]`)),
		withLog("logging in",
			chrome.Click(`button[name="submit"]`)),
		withLog("waiting for page to finish animation",
			wait(loginAnimationTimeout)),
		withLog("extracting session cookie",
			extractCookies(&info)),
	)

	if err := chrome.Run(taskCtx, actions...); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	return &info, nil
}

func extractCookies(info *AuthInfo) chrome.Action {
	return chrome.ActionFunc(func(ctx context.Context) error {
		cookies, err := storage.GetCookies().Do(ctx)
		if err != nil {
			return err
		}

		for _, cookie := range cookies {
			switch cookie.Name {
			case "sid":
				info.SessionID = cookie.Value
			case "XSRF-TOKEN":
				info.XSRFToken = cookie.Value
			}
		}
		return nil
	})
}
