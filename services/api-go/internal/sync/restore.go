package sync

// RestoreText replaces the room's live text with target (F11 history
// restore): delete-all + insert as ordinary live ops, so the broadcast and
// op-log-append subscribers fire untouched — state stays binary updates
// (invariant #1) and history stays append-only (invariant #3). Nothing is
// rewritten; concurrent client edits merge through the CRDT. No-op when the
// text already matches (avoids empty no-op updates in the log).
func RestoreText(room *Room, target string) {
	current := room.Document().Text()
	if current == target {
		return
	}
	if current != "" {
		room.Document().DeleteText(0, len([]rune(current)))
	}
	if target != "" {
		room.Document().InsertText(0, target)
	}
}
