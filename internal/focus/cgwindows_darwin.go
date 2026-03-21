//go:build darwin

package focus

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>

typedef struct {
	int pid;
	int windowNumber;
	int layer;
	char title[512];
	char owner[256];
} CGWindowInfo;

int cg_list_windows(int targetPID, CGWindowInfo* results, int maxResults) {
	CFArrayRef windowList = CGWindowListCopyWindowInfo(kCGWindowListOptionOnScreenOnly, kCGNullWindowID);
	if (!windowList) return 0;

	int count = 0;
	CFIndex total = CFArrayGetCount(windowList);
	for (CFIndex i = 0; i < total && count < maxResults; i++) {
		CFDictionaryRef dict = (CFDictionaryRef)CFArrayGetValueAtIndex(windowList, i);

		int pid = 0;
		CFNumberRef pidRef = CFDictionaryGetValue(dict, kCGWindowOwnerPID);
		if (pidRef) CFNumberGetValue(pidRef, kCFNumberIntType, &pid);

		if (targetPID > 0 && pid != targetPID) continue;

		int layer = -1;
		CFNumberRef layerRef = CFDictionaryGetValue(dict, kCGWindowLayer);
		if (layerRef) CFNumberGetValue(layerRef, kCFNumberIntType, &layer);

		// Only include normal windows (layer 0)
		if (layer != 0) continue;

		int num = 0;
		CFNumberRef numRef = CFDictionaryGetValue(dict, kCGWindowNumber);
		if (numRef) CFNumberGetValue(numRef, kCFNumberIntType, &num);

		results[count].pid = pid;
		results[count].windowNumber = num;
		results[count].layer = layer;
		results[count].title[0] = '\0';
		results[count].owner[0] = '\0';

		CFStringRef ownerRef = CFDictionaryGetValue(dict, kCGWindowOwnerName);
		if (ownerRef) CFStringGetCString(ownerRef, results[count].owner, 256, kCFStringEncodingUTF8);

		CFStringRef nameRef = CFDictionaryGetValue(dict, kCGWindowName);
		if (nameRef) CFStringGetCString(nameRef, results[count].title, 512, kCFStringEncodingUTF8);

		count++;
	}
	CFRelease(windowList);
	return count;
}
*/
import "C"

// cgWindowInfo holds the result of a CGWindowListCopyWindowInfo query.
type cgWindowInfo struct {
	PID          int
	WindowNumber int
	Owner        string
	Title        string
}

// listCGWindows returns all on-screen layer-0 windows for the given PID.
// Uses the CoreGraphics CGWindowListCopyWindowInfo API which reliably sees
// all windows, including Java-based apps (JetBrains IDEs).
func listCGWindows(targetPID int) []cgWindowInfo {
	var results [64]C.CGWindowInfo
	count := C.cg_list_windows(C.int(targetPID), &results[0], 64)

	windows := make([]cgWindowInfo, 0, int(count))
	for i := 0; i < int(count); i++ {
		r := results[i]
		windows = append(windows, cgWindowInfo{
			PID:          int(r.pid),
			WindowNumber: int(r.windowNumber),
			Owner:        C.GoString(&r.owner[0]),
			Title:        C.GoString(&r.title[0]),
		})
	}
	return windows
}
