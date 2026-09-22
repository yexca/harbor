# User Guide

## First Launch

Start Harbor using the [Docker guide](../operations/docker.md), then open
`http://localhost:7750` on the host or the host's reachable address on your device.
The first launch has an empty collection.

Open the top-right **Settings** button and select **Add a service**. Alternatively,
right-click the page, then select the final **+** tile in **All apps**. Enter a
name and address, optionally add a description and icon, then save.

## Home and All Apps

Home shows services with **Show on home** enabled. Select a card to open its
configured address in a new tab.

Right-click the background or a home card to open All apps. Press Space or
`/`, use **Settings → All apps**, or tap the bottom **All apps** launcher for
equivalent access. Text fields retain their normal context menu.
Shift-right-click preserves the browser context menu where the browser supports
it.

All apps shows every saved service. Search by name, description, or address.
Keyboard shortcuts focus the search box. The first match is highlighted; use
the arrow keys to move the highlight and press Enter to open it in a new tab.
Shift with an arrow key still selects text in the search box.
The final plus tile adds a service. Use a card's pencil button to edit it, and
**On home / Show on home** to toggle home visibility. The editor also contains
the visibility checkbox and delete action.

Home selection is saved on the NAS and shared across devices. Hiding a service
does not make its address private. There is no custom ordering or grouping yet.

## Addresses and Icons

Enter an HTTP(S) URL such as `http://media.example:8096`. When a scheme is omitted,
the UI chooses HTTP for local names/IPs and HTTPS for ordinary domain names;
review the result for services using a different scheme. Embedded URL credentials
are rejected. The saved address must be reachable from your browsing device.

**Auto-fetch** requests the website from the NAS and looks for declared favicons,
Apple touch icons, then `/favicon.ico`. You can also upload an image or fetch a
direct image URL. Icons are stored locally; they are not refreshed automatically.

PNG, JPEG, GIF, WebP, ICO, and simple SVG are supported, up to 512 KiB. Active SVG
content is rejected. If fetching fails, keep the initials or upload an image;
an icon is not required to save a service. The NAS must trust the site's TLS
certificate and reach its address. Container `localhost` is not the NAS host.

## Customize Harbor

**Settings → Customize Harbor** changes the browser tab title, the page icon,
and a custom background image. These are saved on the NAS and shared by every
device. Leave the title empty or choose **Use default** to restore Harbor's own
title and icon. The page icon accepts the same formats as service icons.

The custom background accepts JPEG, PNG, GIF, or WebP up to 10 MB. It is uploaded
when you save and becomes the background on that device. Other devices choose
**Custom** under Background in Settings. Uploading a new image replaces the
previous one; **Remove** deletes it, and devices using it return to Mountain.

## Settings and Editing Access

Settings offers System/Light/Dark panels, Mountain/Dusk/Midnight/Custom
backgrounds, and a 12/24-hour clock. These choices are saved in this browser.
Selecting **Custom** before an image exists opens Customize Harbor. The clock uses
device time, updates at minute boundaries, and pauses its timer in hidden tabs.

If the server has an editing password, use **Unlock editing** before changing
services, fetching icons, or customizing Harbor. Viewing the collection remains available. Sessions
expire after 24 hours or a server restart. **Lock editing** ends the session.

Dismiss panels with Escape, the close button, or an outside click. Saving prevents
accidental dismissal; form errors keep the entered values available for correction.
In the service editor, the fields scroll while the title and save/cancel buttons
remain visible. Opening the editor starts at Name without scrolling the page.
Panels use short opening and closing transitions, and appearance choices have a
sliding selection indicator. System reduced-motion settings disable these effects.
Browsers that support reduced transparency use opaque surfaces when requested;
older browsers may dismiss panels immediately instead of animating their exit.

For problems, see [troubleshooting](../operations/troubleshooting.md).
