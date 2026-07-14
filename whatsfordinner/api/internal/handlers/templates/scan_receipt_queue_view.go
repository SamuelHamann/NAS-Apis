package handlers

import "github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"

// QueueStatusOption is one checkbox in the Queue tab's status filter.
type QueueStatusOption struct {
	Value string
	Label string
}

// queueStatusOptions lists every pending_pantry_items.status value, in the
// order its checkbox is shown.
var queueStatusOptions = []QueueStatusOption{
	{store.PendingPantryStatusPending, "Pending"},
	{store.PendingPantryStatusProcessing, "Processing"},
	{store.PendingPantryStatusApproved, "Approved"},
	{store.PendingPantryStatusRejected, "Rejected"},
}

// defaultQueueStatuses is what the Queue tab's checkboxes start checked with
// on a fresh page load (no filter explicitly submitted yet): everything
// except "approved", since an approved item has already become a real
// pantry ingredient and doesn't need to keep showing up here.
var defaultQueueStatuses = []string{
	store.PendingPantryStatusPending,
	store.PendingPantryStatusProcessing,
	store.PendingPantryStatusRejected,
}

// QueueItemView is one row on the Queue tab: a pending_pantry_items row plus
// its pre-computed Tone, so the template stays dumb ("apply this CSS class")
// rather than switching on Status itself.
type QueueItemView struct {
	store.PendingPantryItemRow
	Tone string
}

// queueItemTone maps a pending_pantry_items.status to the row's CSS tone —
// "danger" (red) for rejected, "info" (blue) for processing, "success"
// (green) for approved, and "" (the card's regular color) for pending.
func queueItemTone(status string) string {
	switch status {
	case store.PendingPantryStatusRejected:
		return "danger"
	case store.PendingPantryStatusProcessing:
		return "info"
	case store.PendingPantryStatusApproved:
		return "success"
	default:
		return ""
	}
}

// BuildQueueItems turns store rows into view rows, pulling rejected items to
// the top. Everything else keeps the store's created_at DESC order.
func BuildQueueItems(rows []store.PendingPantryItemRow) []QueueItemView {
	views := make([]QueueItemView, len(rows))
	for i, row := range rows {
		views[i] = QueueItemView{PendingPantryItemRow: row, Tone: queueItemTone(row.Status)}
	}

	rejected := make([]QueueItemView, 0, len(views))
	rest := make([]QueueItemView, 0, len(views))
	for _, v := range views {
		if v.Status == store.PendingPantryStatusRejected {
			rejected = append(rejected, v)
		} else {
			rest = append(rest, v)
		}
	}
	return append(rejected, rest...)
}
