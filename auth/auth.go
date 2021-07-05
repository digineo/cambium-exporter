package auth

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/chromedp/cdproto/network"
	chrome "github.com/chromedp/chromedp"
)

var (
	execPath chrome.ExecAllocatorOption
	headless chrome.ExecAllocatorOption
)

// SetExecPath sets the path to the Chromium or Google Chrome binary.
func SetExecPath(path string) {
	execPath = chrome.ExecPath(path)
}

func SetHeadless(startHeadless bool) {
	headless = chrome.Flag("headless", startHeadless)
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

func simulateTyping(sel interface{}, text string) []chrome.Action {
	actions := make([]chrome.Action, 0, 2*len(text))

	for _, r := range text {
		actions = append(actions,
			chrome.SendKeys(sel, string(r)),
			wait(time.Duration(25+rand.Intn(30))*time.Millisecond),
		)
	}

	return actions
}

func Login(ctx context.Context, instanceUrl, username, password string) (*AuthInfo, error) {
	opts := chrome.DefaultExecAllocatorOptions[:]
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

	actions := []chrome.Action{
		chrome.Navigate(instanceUrl),
		chrome.WaitVisible(`a[href="/cn-rtr/sso"]`),
		chrome.Click(`a[href="/cn-rtr/sso"]`, chrome.NodeVisible),
		chrome.WaitVisible(`form#login`),
	}
	actions = append(actions, simulateTyping(`input[name="email"]`, username)...)
	actions = append(actions, simulateTyping(`input[name="password"]`, password)...)
	actions = append(actions,
		chrome.Click(`input[name="remember"]`),
		chrome.Click(`button[type="submit"]`),
		wait(5*time.Second),
	)

	info := AuthInfo{}
	actions = append(actions, extractCookies(&info))

	if err := chrome.Run(taskCtx, actions...); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	return &info, nil
}

func extractCookies(info *AuthInfo) chrome.Action {
	return chrome.ActionFunc(func(ctx context.Context) error {
		cookies, err := network.GetAllCookies().Do(ctx)
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
