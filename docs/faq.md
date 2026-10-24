---
title: FAQ
---

# Frequently Asked Questions

## Technical

#### Why cannot I copy an entire GIF?

Due to the limitation of [Clipboard API](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard_API), the only formats available are `text/plain`, `text/html` and `image/png`. (see [`ClipboardItem.supports()`](https://developer.mozilla.org/en-US/docs/Web/API/ClipboardItem/supports_static))  
This limitation helps improve security by forbidding some files that may be harmful to system, but also makes it impossible to copy other formats.  

Some apps, however, allow the user to paste GIFs by encoding the GIF as data urls and wrapping it into a `<img>` element. For these apps, we have a _Copy Mode_ settings item for you to opt into. _HTML_ mode always wraps images into an `<img>` element, while _PNG_ mode draws the image onto a canvas and copies the converted image. _Auto_ mode uses _HTML_ only for `.gif` images, and _PNG_ for other images.

#### Why some GIFs become unanimated after uploading?

It's been tested that some software cannot correctly pass file types when dragging files from inside. In this case, the browser will not receive the correct file format.  

Currently there's no way of solving this. We suggest saving the image as gif files and dragging the file from your system explorer.

#### Is it possible to change the image?

No.

#### Cannot find newly-uploaded images via searching?

Indexing is done via a background job queue on the server side and should complete within a few seconds after uploading. Wait for a while and try again.

## Content

### Image eligibility

#### How to decide which duplicate shall be deleted?

In brief, the one with lower quality and more content unrelated to LONG is tend to be deleted.

- For images that have no apparent differences in content, resolution and definition, one that looks better is likely to be kept. For example, ones with more natural saturation or hue.
- For images that have the same content, but differ in resolution and definition (see below), the one that minimizes resolution and maximizez definition will be kept.
- For images obtained by cropping a portion out of a larger image, the larger one will be kept.
- For static images taken from a frame of a GIF, the GIF will be kept, except for when the static image is modified to a proper extent.

#### Should I upload images not directly containing LONG faces?

It depends. Images that do not directly contain LONG faces but satisfy any of the conditions below can still be considered eligible for upload:
1. Belongs to an existing variant series.
2. Mimics LONG faces with other elements (like a [socket](https://longhub.top/post/89d4b854-5ff1-48ae-bec7-ea6aff29923d)).
3. Is obtained by passing a LONG image through some sort of transformation, usually somewhat related to AI. (like the [output image](https://longhub.top/post/bd9acd7a-7d44-4623-bebf-b3294da30a3e) after feeding LONG face into a Stable Diffusion img2img, or an [AI upscaled image](https://longhub.top/post/4b8266cc-7b1e-408f-b3c3-6dc6f92cd7dd))

Here, "directly" containing LONG faces refers to that the image contains at least a part of a LONG face that is sufficient to recognize.

For images satisfying condition 1 and 2, add a `longless` tag when uploading.
