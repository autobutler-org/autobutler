# Taking over a smart frame or tablet: what you take on

Some people want to turn a smart photo frame or a cheap tablet they already own into a screen for their Quark:
a slideshow of the family's photos, a calendar, or music through a Bluetooth speaker. Doing that usually means
changing the software the maker put on the device. This guide covers what you risk when you do that, and how
to keep yourself, your family and your home network safe.

It is not a set of instructions for any particular device, and it does not promise that any device will work.
Quark does not sell, ship or support changes to other companies' hardware. You make those changes yourself, at
your own risk.

## Before you start: is it worth it?

Ask yourself two questions first.

- **Could you live without this device if it stopped working for good?** Changing a device's software can
  leave it unable to turn on or start up properly. People call this "bricking", because the device ends up
  about as useful as a brick. Sometimes it can be fixed, and sometimes it cannot.
- **Are you willing to give up the warranty?** Most makers treat changing the device's software as a reason
  to refuse repairs or returns, even for faults that have nothing to do with the change. Check the warranty
  and return window before you begin. If the device is new and you are not sure, it may be better to wait
  until the return window has passed, or not to start at all.

If the answer to either is no, leave the device as it is.

## Keep a backup you have checked

Before anything on the device is changed, make a full copy of what is on it now. This copy is often called a
"dump" or an "image". It is your only way back if something goes wrong.

- **Make the backup before the first change, not partway through.** Once something on the device has been
  overwritten, a backup taken afterward cannot bring the original back.
- **Check that the backup is complete and readable.** A backup that was cut short or is damaged is no backup.
  Whatever tool made it should be able to confirm the copy matches the device. If you cannot confirm that,
  treat it as if you have no backup.
- **Keep at least two copies, in two places.** For example, one on your computer and one on a USB drive kept
  in a drawer. One copy on one disk is one accident away from gone.
- **Write down which device each backup came from.** Frames sold under the same name can have different parts
  inside, and a backup from one unit may not suit another, even an identical-looking one.

If anyone helping you suggests skipping the backup "just this once", stop.

## Keep your backup private

A full backup of a device holds much more than photos. It can include:

- the name and password of every Wi-Fi network the device has joined;
- the device's serial number and other numbers that identify it to its maker;
- the account the device was signed in to, and sometimes what is needed to sign in to it again;
- photos, contacts or messages that were on the device.

Treat the backup like a spare house key.

- **Do not post it, share it or upload it** to a forum, a chat group, a public file-sharing site or an
  online issue tracker, even to ask for help. If someone asks to see part of it, share only that part, and
  read it first.
- **Store it where only you can get at it.** Your own home folder on your Quark is a reasonable place, as long
  as you remember that anyone who is an admin on that Quark can open it too. A shared folder that other people
  can open is not a good place.
- **If you think a copy has leaked, change your Wi-Fi password** and the password of the account the device
  used, even if you are not sure.

## Turn off the maker's automatic updates before it goes online

Many devices check with their maker for new software as soon as they reach the internet, and install it on
their own.

- An update can undo your changes, lock the device so they cannot be made again, or, if it arrives while the
  device is partly changed, leave it unable to start.
- An update can also bring back the maker's app and whatever it sends home, such as how the device is used.

So, before you connect a changed device to Wi-Fi:

- turn off automatic updates in the device's settings, if it has that option;
- if the maker's app is what installs updates, turn that app off or remove it;
- if you are not sure updates are off, keep the device off Wi-Fi until you are.

Keep in mind that turning off the maker's updates also means you stop getting the maker's security fixes. That
is part of the trade. A device that no longer gets fixes should only reach what it needs to, which the next
section is about.

## What a changed device can reach on your home network

Taking full control of a device (often called "rooting" it) removes the limits the maker put on what its
software is allowed to do. That cuts both ways: you can run what you want on it, and so can anything harmful
that ends up on it.

Once it is on your Wi-Fi, a device like this can, in principle:

- see the other devices on your home network, such as computers, phones, printers, cameras and your Quark;
- try to connect to any of them that accept connections from inside your home;
- listen with its microphone or watch with its camera, if it has them;
- send anything it can reach out to the internet.

None of this happens on its own. It becomes possible if the device runs something you did not mean it to:
an app from an unknown source, a leftover piece of the maker's software, or a tool someone recommended without
saying what it does.

Some ways to keep the risk small:

- **Install only what you need**, from sources you trust, and remove what you do not use.
- **Put it on a guest network** if your router offers one, so it can reach the internet but not your other
  devices. If it needs to reach your Quark, check whether your router lets a guest-network device do that
  before you rely on it.
- **Sign it in to Quark with an ordinary account, never an admin account.** An admin account can see and
  change everything on your Quark, including other people's files. See
  [Accounts and sharing](./accounts-and-sharing.md) for the difference between the two. A screen that shows a
  slideshow needs no more than that.
- **Cover or switch off the camera and microphone** if the device has them and you do not need them.
- **Know how to unplug it.** If the device starts acting oddly, unplug it and take it off your Wi-Fi first,
  and work out what happened second.

## If something goes wrong

- **Stop and do not keep trying things.** Each new attempt can overwrite more of what was there. Unplug the
  device and leave it.
- **Find your checked backup** and the note of which device it came from.
- **Ask for help without sharing your backup.** Describe what you did and what the device does now. Never
  send the backup itself to someone you do not know.
- **Be ready for it not to come back.** Even with a good backup, some devices cannot be restored. That is the
  risk you accepted at the start.

## In short

1. Only change a device you can afford to lose, and expect the warranty to end.
2. Make a full backup before the first change, check it, and keep two copies.
3. Keep that backup private: it holds your Wi-Fi password and the device's serial number.
4. Turn off the maker's automatic updates before the device goes online.
5. Give the device as little access as it needs: a guest network if you can, and an ordinary Quark account,
   never an admin one.
