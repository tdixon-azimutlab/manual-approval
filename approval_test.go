package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-github/v43/github"
)

func TestNewApprovalEnvironment_WithLabels(t *testing.T) {
	labels := []string{"bug", "enhancement", "help wanted"}

	apprv, err := newApprovalEnvironment(
		nil,
		"owner/repo",
		"owner",
		12345,
		[]string{"approver1"},
		1,
		"Test Issue",
		"Test Body",
		labels,
		"owner",
		"repo",
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(apprv.issueLabels) != len(labels) {
		t.Fatalf("expected %d labels but got %d", len(labels), len(apprv.issueLabels))
	}

	for i, label := range apprv.issueLabels {
		if label != labels[i] {
			t.Fatalf("expected label %q but got %q at index %d", labels[i], label, i)
		}
	}
}

func TestNewApprovalEnvironment_WithoutLabels(t *testing.T) {
	apprv, err := newApprovalEnvironment(
		nil,
		"owner/repo",
		"owner",
		12345,
		[]string{"approver1"},
		1,
		"Test Issue",
		"Test Body",
		[]string{},
		"owner",
		"repo",
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(apprv.issueLabels) != 0 {
		t.Fatalf("expected 0 labels but got %d", len(apprv.issueLabels))
	}
}

func TestCreateApprovalIssue_WithLabels(t *testing.T) {
	expectedLabels := []string{"bug", "enhancement"}
	var capturedRequest github.IssueRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/issues") {
			if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			issue := &github.Issue{
				Number:  github.Int(1),
				HTMLURL: github.String("https://github.com/owner/repo/issues/1"),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(issue)
			return
		}
		// Handle comment creation
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/comments") {
			comment := &github.IssueComment{ID: github.Int64(1)}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(comment)
			return
		}
	}))
	defer server.Close()

	client, err := github.NewEnterpriseClient(server.URL+"/", server.URL+"/", nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	apprv := &approvalEnvironment{
		client:          client,
		repoFullName:    "owner/repo",
		repo:            "repo",
		repoOwner:       "owner",
		runID:           12345,
		issueApprovers:  []string{"approver1"},
		issueLabels:     expectedLabels,
		targetRepoOwner: "owner",
		targetRepoName:  "repo",
	}

	err = apprv.createApprovalIssue(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedRequest.Labels == nil {
		t.Fatal("expected labels to be set in request but was nil")
	}

	if len(*capturedRequest.Labels) != len(expectedLabels) {
		t.Fatalf("expected %d labels but got %d", len(expectedLabels), len(*capturedRequest.Labels))
	}

	for i, label := range *capturedRequest.Labels {
		if label != expectedLabels[i] {
			t.Fatalf("expected label %q but got %q at index %d", expectedLabels[i], label, i)
		}
	}
}

func TestCreateApprovalIssue_WithoutLabels(t *testing.T) {
	var capturedRequest github.IssueRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/issues") {
			if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			issue := &github.Issue{
				Number:  github.Int(1),
				HTMLURL: github.String("https://github.com/owner/repo/issues/1"),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(issue)
			return
		}
		// Handle comment creation
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/comments") {
			comment := &github.IssueComment{ID: github.Int64(1)}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(comment)
			return
		}
	}))
	defer server.Close()

	client, err := github.NewEnterpriseClient(server.URL+"/", server.URL+"/", nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	apprv := &approvalEnvironment{
		client:          client,
		repoFullName:    "owner/repo",
		repo:            "repo",
		repoOwner:       "owner",
		runID:           12345,
		issueApprovers:  []string{"approver1"},
		issueLabels:     []string{},
		targetRepoOwner: "owner",
		targetRepoName:  "repo",
	}

	err = apprv.createApprovalIssue(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedRequest.Labels != nil {
		t.Fatalf("expected labels to be nil but got %v", *capturedRequest.Labels)
	}
}

func TestApprovalFromComments(t *testing.T) {
	login1 := "login1"
	login2 := "login2"
	login3 := "login3"
	bodyApproved := "Approved"
	bodyDenied := "Denied"
	bodyPending := "not approval or denial"

	login1u := strings.ToUpper(login1)

	testCases := []struct {
		name             string
		comments         []*github.IssueComment
		approvers        []string
		minimumApprovals int
		expectedStatus   approvalStatus
	}{
		{
			name: "single_approver_single_comment_approved",
			comments: []*github.IssueComment{
				{
					User: &github.User{Login: &login1},
					Body: &bodyApproved,
				},
			},
			approvers:      []string{login1},
			expectedStatus: approvalStatusApproved,
		},
		{
			name: "single_approver_single_comment_denied",
			comments: []*github.IssueComment{
				{
					User: &github.User{Login: &login1},
					Body: &bodyDenied,
				},
			},
			approvers:      []string{login1},
			expectedStatus: approvalStatusDenied,
		},
		{
			name: "single_approver_single_comment_pending",
			comments: []*github.IssueComment{
				{
					User: &github.User{Login: &login1},
					Body: &bodyPending,
				},
			},
			approvers:      []string{login1},
			expectedStatus: approvalStatusPending,
		},
		{
			name: "single_approver_multi_comment_approved",
			comments: []*github.IssueComment{
				{
					User: &github.User{Login: &login1},
					Body: &bodyPending,
				},
				{
					User: &github.User{Login: &login1},
					Body: &bodyApproved,
				},
			},
			approvers:      []string{login1},
			expectedStatus: approvalStatusApproved,
		},
		{
			name: "multi_approver_approved",
			comments: []*github.IssueComment{
				{
					User: &github.User{Login: &login1},
					Body: &bodyApproved,
				},
				{
					User: &github.User{Login: &login2},
					Body: &bodyApproved,
				},
			},
			approvers:      []string{login1, login2},
			expectedStatus: approvalStatusApproved,
		},
		{
			name: "multi_approver_mixed",
			comments: []*github.IssueComment{
				{
					User: &github.User{Login: &login1},
					Body: &bodyPending,
				},
				{
					User: &github.User{Login: &login2},
					Body: &bodyApproved,
				},
			},
			approvers:      []string{login1, login2},
			expectedStatus: approvalStatusPending,
		},
		{
			name: "multi_approver_denied",
			comments: []*github.IssueComment{
				{
					User: &github.User{Login: &login1},
					Body: &bodyDenied,
				},
				{
					User: &github.User{Login: &login2},
					Body: &bodyApproved,
				},
			},
			approvers:      []string{login1, login2},
			expectedStatus: approvalStatusDenied,
		},
		{
			name: "multi_approver_minimum_one_approval",
			comments: []*github.IssueComment{
				{
					User: &github.User{Login: &login1},
					Body: &bodyPending,
				},
				{
					User: &github.User{Login: &login2},
					Body: &bodyApproved,
				},
			},
			approvers:        []string{login1, login2},
			expectedStatus:   approvalStatusApproved,
			minimumApprovals: 1,
		},
		{
			name: "multi_approver_minimum_two_approvals",
			comments: []*github.IssueComment{
				{
					User: &github.User{Login: &login1},
					Body: &bodyApproved,
				},
				{
					User: &github.User{Login: &login2},
					Body: &bodyApproved,
				},
			},
			approvers:        []string{login1, login2, login3},
			expectedStatus:   approvalStatusApproved,
			minimumApprovals: 2,
		},
		{
			name: "multi_approver_approvals_less_than_minimum",
			comments: []*github.IssueComment{
				{
					User: &github.User{Login: &login1},
					Body: &bodyApproved,
				},
			},
			approvers:        []string{login1, login2, login3},
			expectedStatus:   approvalStatusPending,
			minimumApprovals: 2,
		},
		{
			name: "single_approver_single_comment_approved_case_insensitive",
			comments: []*github.IssueComment{
				{
					User: &github.User{Login: &login1u},
					Body: &bodyApproved,
				},
			},
			approvers:      []string{login1},
			expectedStatus: approvalStatusApproved,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := approvalFromComments(testCase.comments, testCase.approvers, testCase.minimumApprovals)
			if err != nil {
				t.Fatalf("error getting approval from comments: %v", err)
			}

			if actual != testCase.expectedStatus {
				t.Fatalf("actual %s, expected %s", actual, testCase.expectedStatus)
			}
		})
	}
}

func TestApprovedCommentBody(t *testing.T) {
	testCases := []struct {
		name               string
		commentBody        string
		isSuccess          bool
		customApprovalWord string
	}{
		{
			name:               "approved_lowercase_no_punctuation",
			commentBody:        "approved",
			isSuccess:          true,
			customApprovalWord: "",
		},
		{
			name:               "approve_lowercase_no_punctuation",
			commentBody:        "approve",
			isSuccess:          true,
			customApprovalWord: "",
		},
		{
			name:               "lgtm_lowercase_no_punctuation",
			commentBody:        "lgtm",
			isSuccess:          true,
			customApprovalWord: "",
		},
		{
			name:               "yes_lowercase_no_punctuation",
			commentBody:        "yes",
			isSuccess:          true,
			customApprovalWord: "",
		},
		{
			name:               "approve_uppercase_no_punctuation",
			commentBody:        "APPROVE",
			isSuccess:          true,
			customApprovalWord: "",
		},
		{
			name:               "approved_titlecase_period",
			commentBody:        "Approved.",
			isSuccess:          true,
			customApprovalWord: "",
		},
		{
			name:               "approved_titlecase_exclamation",
			commentBody:        "Approved!",
			isSuccess:          true,
			customApprovalWord: "",
		},
		{
			name:               "approved_titlecase_multi_exclamation",
			commentBody:        "Approved!!",
			isSuccess:          true,
			customApprovalWord: "",
		},
		{
			name:               "approved_titlecase_question",
			commentBody:        "Approved?",
			isSuccess:          false,
			customApprovalWord: "",
		},
		{
			name:               "sentence_with_keyword",
			commentBody:        "should i approve this",
			isSuccess:          false,
			customApprovalWord: "",
		},
		{
			name:               "sentence_without_keyword",
			commentBody:        "this is just some random comment",
			isSuccess:          false,
			customApprovalWord: "",
		},
		{
			name:               "approved_with_newline",
			commentBody:        "approved\n",
			isSuccess:          true,
			customApprovalWord: "",
		},
		{
			name:               "approved_with_exclamation_newline",
			commentBody:        "approved!\n",
			isSuccess:          true,
			customApprovalWord: "",
		},
		{
			name:               "approved_with_multi_exclamation_multi_newline",
			commentBody:        "approved!!!\n\n\n",
			isSuccess:          true,
			customApprovalWord: "",
		},
		{
			name:               "approved_with_custom_approval_word",
			commentBody:        "shipit",
			isSuccess:          true,
			customApprovalWord: "shipit",
		},
		{
			name:               "approved_with_github_emoji_syntax",
			commentBody:        ":shipit:",
			isSuccess:          true,
			customApprovalWord: ":shipit:",
		},
		{
			name:               "approved_with_custom_hashtag",
			commentBody:        "#shipit",
			isSuccess:          true,
			customApprovalWord: "#shipit",
		},
		{
			name:               "approved_with_actual_emoji_✅",
			commentBody:        "✅ ",
			isSuccess:          true,
			customApprovalWord: "✅",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// before each
			word := testCase.customApprovalWord
			if len(word) > 0 {
				approvedWords = append(approvedWords, word)
			}

			// test
			actual, err := isApproved(testCase.commentBody)
			if err != nil {
				t.Fatalf("error getting approval: %v", err)
			}
			if actual != testCase.isSuccess {
				t.Fatalf("expected %v but got %v", testCase.isSuccess, actual)
			}

			// after each
			if len(word) > 0 {
				approvedWords = approvedWords[:len(approvedWords)-1]
			}
		})
	}
}

func TestDeniedCommentBody(t *testing.T) {
	testCases := []struct {
		name             string
		commentBody      string
		isSuccess        bool
		customDenialWord string
	}{
		{
			name:             "denied_lowercase_no_punctuation",
			commentBody:      "denied",
			isSuccess:        true,
			customDenialWord: "",
		},
		{
			name:             "deny_lowercase_no_punctuation",
			commentBody:      "deny",
			isSuccess:        true,
			customDenialWord: "",
		},
		{
			name:             "no_lowercase_no_punctuation",
			commentBody:      "no",
			isSuccess:        true,
			customDenialWord: "",
		},
		{
			name:             "deny_uppercase_no_punctuation",
			commentBody:      "DENY",
			isSuccess:        true,
			customDenialWord: "",
		},
		{
			name:             "denied_titlecase_period",
			commentBody:      "Denied.",
			isSuccess:        true,
			customDenialWord: "",
		},
		{
			name:             "denied_titlecase_exclamation",
			commentBody:      "Denied!",
			isSuccess:        true,
			customDenialWord: "",
		},
		{
			name:             "deny_titlecase_question",
			commentBody:      "Deny?",
			isSuccess:        false,
			customDenialWord: "",
		},
		{
			name:             "sentence_with_keyword",
			commentBody:      "should i deny this",
			isSuccess:        false,
			customDenialWord: "",
		},
		{
			name:             "sentence_without_keyword",
			commentBody:      "this is just some random comment",
			isSuccess:        false,
			customDenialWord: "",
		},
		{
			name:             "denied_with_newline",
			commentBody:      "denied\n",
			isSuccess:        true,
			customDenialWord: "",
		},
		{
			name:             "denied_with_custom_word",
			commentBody:      "naw",
			isSuccess:        true,
			customDenialWord: "naw",
		},
		{
			name:             "denied_with_github_emoji",
			commentBody:      ":no_entry_sign: ",
			isSuccess:        true,
			customDenialWord: ":no_entry_sign:",
		},
		{
			name:             "denied_with_hashtag",
			commentBody:      "#noway",
			isSuccess:        true,
			customDenialWord: "#noway",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// before each
			word := testCase.customDenialWord
			if len(word) > 0 {
				deniedWords = append(deniedWords, word)
			}

			// test
			actual, err := isDenied(testCase.commentBody)
			if err != nil {
				t.Fatalf("error getting approval: %v", err)
			}
			if actual != testCase.isSuccess {
				t.Fatalf("expected %v but got %v", testCase.isSuccess, actual)
			}

			// after each
			if len(word) > 0 {
				deniedWords = deniedWords[:len(deniedWords)-1]
			}
		})
	}
}

func TestSaveOutput(t *testing.T) {
	testCases := []struct {
		name                string
		approvalIssueNumber int
		env_github_output   string
		isSuccess           bool
	}{
		{
			name:                "save_output_with_env",
			approvalIssueNumber: 123,
			env_github_output:   "./output.txt",
			isSuccess:           true,
		},
		{
			name:                "fail_save_output_without_env",
			approvalIssueNumber: 123,
			env_github_output:   "",
			isSuccess:           false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("GITHUB_OUTPUT", testCase.env_github_output)
			a := approvalEnvironment{
				client:              nil,
				repoFullName:        "",
				repo:                "",
				repoOwner:           "",
				runID:               -1,
				approvalIssueNumber: testCase.approvalIssueNumber,
				issueTitle:          "",
				issueBody:           "",
				issueApprovers:      nil,
				minimumApprovals:    0,
			}

            if err := os.Remove(testCase.env_github_output); err != nil && !os.IsNotExist(err) {
                t.Fatalf("failed to remove file: %v", err)
            }

			actual, err := a.SetActionOutputs(nil)

			if err != nil {
				t.Fatalf("error creating output file: %v: %v", testCase.env_github_output, err)
			}

			if actual != testCase.isSuccess {
				t.Fatalf("expected %v but got %v", testCase.isSuccess, actual)
			}

			if actual == true {
				if _, err := os.Stat(testCase.env_github_output); errors.Is(err, os.ErrNotExist) {
					t.Fatalf("expected create output file %v but it was not", testCase.env_github_output)
				}
			}
		})
	}
}
