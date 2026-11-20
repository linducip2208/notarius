// Package gov implements ASSISTED government workflows (AHU/SABH, OSS).
// There are no official public endpoints embedded here: the module prepares
// data, tracks manual submission checklists, and archives receipts/evidence.
// Any future official API adapter must implement Submitter.
package gov

import (
	"errors"
	"time"
)

// Channel is the government system involved.
type Channel string

const (
	ChannelAHU Channel = "AHU"
	ChannelOSS Channel = "OSS"
)

// Status of an assisted submission.
type Status string

const (
	StatusDraft     Status = "DRAFT"
	StatusReady     Status = "READY"
	StatusSubmitted Status = "SUBMITTED"
	StatusAccepted  Status = "ACCEPTED"
	StatusRejected  Status = "REJECTED"
)

// Submission tracks one assisted filing prepared by the office and
// filed manually by authorized staff on the official portal.
type Submission struct {
	ID          int64
	Channel     Channel
	Reference   string // office reference, e.g. company name + matter
	Status      Status
	Checklist   []CheckItem
	ReceiptNo   string
	ReceiptFile string
	Notes       string
	UpdatedAt   time.Time
}

// CheckItem is one manual step (e.g. "cek nama perseroan di SABH").
type CheckItem struct {
	Label string
	Done  bool
}

// NewSubmission creates a draft with the standard checklist for a channel.
func NewSubmission(ch Channel, ref string) Submission {
	var items []string
	switch ch {
	case ChannelAHU:
		items = []string{
			"Cek nama perseroan (SABH)",
			"Siapkan data pendiri & modal",
			"Siapkan susunan direksi & komisaris",
			"Upload akta ke pemohon untuk persetujuan",
			"Input manual ke portal AHU oleh staf berwenang",
			"Arsipkan bukti penerimaan / SK",
		}
	case ChannelOSS:
		items = []string{
			"Siapkan NIB & data KBLI",
			"Verifikasi data perusahaan",
			"Input manual ke portal OSS oleh staf berwenang",
			"Arsipkan NIB / izin terbit",
		}
	}
	s := Submission{Channel: ch, Reference: ref, Status: StatusDraft, UpdatedAt: time.Now().UTC()}
	for _, it := range items {
		s.Checklist = append(s.Checklist, CheckItem{Label: it})
	}
	return s
}

// Advance moves status forward; backward moves are rejected to keep
// an honest evidence trail.
func (s *Submission) Advance(to Status) error {
	order := map[Status]int{
		StatusDraft: 0, StatusReady: 1, StatusSubmitted: 2, StatusAccepted: 3, StatusRejected: 3,
	}
	if order[to] < order[s.Status] {
		return errors.New("cannot move submission backwards; create a new revision instead")
	}
	s.Status = to
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// Complete reports whether every checklist item is done.
func (s *Submission) Complete() bool {
	for _, c := range s.Checklist {
		if !c.Done {
			return false
		}
	}
	return true
}
