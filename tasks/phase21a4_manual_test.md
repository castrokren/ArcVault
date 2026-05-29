# Phase 21a-4 Manual Testing Checklist

**Date:** 2026-05-29  
**Phase:** 21a-4 (Job Detail Modal with Full Logs Display)  
**Tester:** [Your name]

---

## Pre-Test Setup

### Build the project
```powershell
cd C:\Projects\ArcVault2.0
go build -o coordinator.exe ./cmd/coordinator
```

### Start the coordinator
```powershell
./coordinator.exe start
# Should see: "listening on :8080"
# Dashboard available at: http://localhost:8080
```

---

## Test Suite

### 1. Modal Opens on Job Row Click ✓/✗
**Steps:**
1. Open dashboard at http://localhost:8080
2. Navigate to Jobs page
3. Click any job row (not the delete button)

**Expected:** Modal opens with fade-in animation, shows job details in header

**Notes:**
- Job ID should be displayed in mono font
- Status badge should show (running/completed/failed/pending)
- Close button (✕) visible in top-right
- Download button visible next to close

---

### 2. Logs Load on Modal Open ✓/✗
**Steps:**
1. Open modal for a job with logs (if none exist, create and run a job first)
2. Wait for logs to load

**Expected:** Logs appear with line numbers on left, log text on right

**Verification:**
- [ ] Logs display in monospace font
- [ ] Line numbers are right-aligned, grayed out
- [ ] First log line number matches expected (e.g., 1, 26, etc. based on pagination)
- [ ] Logs are in chronological order (oldest first)
- [ ] No logs: "No logs available" message shows

---

### 3. Pagination Works ✓/✗
**Steps:**
1. Open modal for job with 100+ logs
2. Note "Showing X–Y of Z logs" at bottom
3. Click "Next →" button
4. Verify logs change

**Expected:** 
- First page: logs 1-50 (or whatever limit)
- Second page: logs 51-100
- Page indicator updates: "Page 2 of 3"
- Prev/Next buttons enable/disable appropriately

**Verification:**
- [ ] Pagination info updates on page change
- [ ] First log on page 2 is `log line 51` (or expected first line)
- [ ] Last page has fewer items if total not divisible by 50
- [ ] Prev button disabled on page 1
- [ ] Next button disabled on last page
- [ ] Can navigate back and forth

---

### 4. Custom Pagination Limit ✓/✗
**Steps:**
1. Manually edit URL: `http://localhost:8080/#/jobs`
2. Check browser console for network requests
3. Verify API call includes `?page=1&limit=25`

**Expected:** Default limit is 25 logs per page

**Verification:**
- [ ] API shows `limit=25` in request
- [ ] Pagination shows "Page X of Y" with correct page count

---

### 5. Real-Time WebSocket Updates ✓/✗
**Steps:**
1. Open modal for a running job
2. Look for "Live updates" indicator at bottom of logs
3. Trigger progress update (or wait for natural update)
4. Watch for new logs to appear

**Expected:** 
- Live indicator appears with pulsing dot
- New logs stream in with green highlight + slide animation
- Live indicator disappears after 2 seconds

**Verification:**
- [ ] Green "Live updates" badge visible
- [ ] New logs appear with animation
- [ ] Logs appear in chronological order
- [ ] If on last page, automatically loads new logs

---

### 6. Download Logs Feature ✓/✗
**Steps:**
1. Open modal
2. Click "⬇ Download" button
3. Check Downloads folder

**Expected:** File named `{jobId}-logs.txt` appears with all logs

**Verification:**
- [ ] File downloads to Downloads folder
- [ ] Filename format: `{jobId}-logs.txt` (e.g., `job-123-logs.txt`)
- [ ] File opens in text editor
- [ ] Contains all visible logs, one per line
- [ ] Logs are in chronological order

---

### 7. Scroll Performance ✓/✗
**Steps:**
1. Open modal for job with 500+ logs
2. Scroll through logs (up/down)
3. Monitor for lag or jank

**Expected:** Smooth scrolling, no performance issues

**Verification:**
- [ ] Scrolling is smooth
- [ ] Custom scrollbar visible and functional
- [ ] No lag when scrolling through large logs
- [ ] Hover effects work smoothly

---

### 8. Modal Close ✓/✗
**Steps:**
1. Open modal
2. Click close button (✕) in top-right
3. Try clicking outside modal on backdrop

**Expected:** Modal closes smoothly, backdrop fades out

**Verification:**
- [ ] Close button closes modal
- [ ] Clicking backdrop closes modal
- [ ] Fade-out animation plays
- [ ] Jobs page visible again

---

### 9. Edge Cases ✓/✗

#### 9a. No Logs
**Steps:**
1. Create a new job, don't run it
2. Click to open modal

**Expected:** "No logs available" message

**Verification:**
- [ ] Message displays clearly
- [ ] No errors in console

#### 9b. Job with 1 Log
**Steps:**
1. Find or create job with exactly 1 log
2. Open modal

**Expected:** 1 log displays, pagination shows "Page 1 of 1"

**Verification:**
- [ ] Single log displays
- [ ] Pagination disabled (no Next/Prev buttons)

#### 9c. Very Long Log Line (1000+ chars)
**Steps:**
1. Trigger progress update with long log line
2. Scroll horizontally

**Expected:** Long lines wrap or scroll without breaking layout

**Verification:**
- [ ] Long lines wrap properly
- [ ] Modal doesn't break
- [ ] Horizontal scroll works if needed

---

### 10. Mobile Responsiveness ✓/✗
**Steps:**
1. Open DevTools (F12)
2. Toggle device emulation (mobile view)
3. Open modal on mobile viewport

**Expected:** Modal scales to fit screen

**Verification:**
- [ ] Modal visible and usable on mobile
- [ ] Logs readable on small screen
- [ ] Pagination controls usable
- [ ] Close button accessible

---

### 11. Accessibility ✓/✗
**Steps:**
1. Open DevTools → Accessibility tab
2. Check for accessibility issues
3. Try keyboard navigation (Tab key)

**Expected:** No major accessibility violations

**Verification:**
- [ ] No console errors related to accessibility
- [ ] Buttons focus with Tab key
- [ ] ARIA labels present on interactive elements

---

### 12. Error Handling ✓/✗

#### 12a. Network Error During Log Load
**Steps:**
1. Open DevTools → Network tab
2. Throttle to "Offline"
3. Open modal
4. Restore connection

**Expected:** Error message shows, retry works

**Verification:**
- [ ] Error displayed gracefully
- [ ] Page recovers when connection restored

#### 12b. WebSocket Disconnection
**Steps:**
1. Open modal for running job
2. Disable network or close DevTools
3. Re-enable network

**Expected:** WebSocket reconnects automatically

**Verification:**
- [ ] Live updates resume after reconnect
- [ ] No page reload needed

---

## Summary

**Total Tests:** 12 categories  
**Passed:** ___/12  
**Failed:** ___/12  
**Blockers:** None / [list]  

**Notes:**
```
[Add any observations, issues, or edge cases discovered]
```

**Signed Off By:** _________________  
**Date:** _________________

---

## If Any Fail

1. **Record the failure:** Describe exactly what happened vs. what was expected
2. **Check the console:** DevTools → Console tab for error messages
3. **Check Network tab:** Look for failed API requests
4. **Report back:** Paste screenshots + console errors + network details

---

## Quick Test (If Short on Time)

Run just these core tests:
1. ✓ Modal opens on job click
2. ✓ Logs load and display
3. ✓ Pagination works (Next/Prev buttons)
4. ✓ Download button works
5. ✓ Modal closes properly

If all 5 pass, Phase 21a-4 is verified.
