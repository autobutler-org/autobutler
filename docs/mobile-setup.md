# Running AutoButler on a Physical Device

The AutoButler Flutter app runs on Android and iOS. Here's how to get it on your phone.

---

## Before you start

You need:
- Flutter set up locally (`make setup/flutter`)
- A butler running on your local network
- Your phone on the same Wi-Fi network as the butler

---

## Android

### 1. Enable developer mode on your phone

Settings → About Phone → tap **Build Number** 7 times. Then go to Settings → Developer Options → enable **USB Debugging**.

### 2. Connect via USB and trust the computer

Plug in your phone. When it asks "Allow USB debugging?", tap Allow.

Verify it's detected:
```bash
flutter devices
```

You should see your phone listed.

### 3. Run the app

```bash
flutter run
```

Flutter will list connected devices and prompt you to pick one if more than one is attached.

Or build an APK and install it manually:
```bash
make build/frontend/android        # produces build/app/outputs/flutter-apk/app-debug.apk
adb install build/app/outputs/flutter-apk/app-debug.apk
```

---

## iOS

iOS requires macOS + Xcode. This won't work from Linux.

### 1. Set up the iOS environment (macOS only)

```bash
make setup/ios
```

This installs CocoaPods, runs Xcode first-launch setup, and downloads the iOS platform.

### 2. Register your device in Xcode

Open Xcode → Window → Devices and Simulators → plug in your iPhone → click "Trust". Xcode needs to register the device UDID before you can run on it.

You'll also need a free or paid Apple Developer account. A free account lets you sideload for 7 days before needing to re-sign.

### 3. Run the app

```bash
flutter run
```

Or build without code signing (for manual distribution):
```bash
make build/frontend/ios    # flutter build ios --debug --no-codesign
```

Note: `--no-codesign` means the app can't be installed directly — you need to sign it via Xcode or a provisioning profile. For dev purposes, just use `flutter run` with the device connected.

---

## Connecting to your butler

Once the app is running, open Settings and add your butler's local IP as the host:

```
http://192.168.x.x:8080
```

Find the IP by checking your router's device list, or running `hostname -I` on the Pi.

The app communicates entirely over your local network — no cloud relay. Make sure your phone and butler are on the same Wi-Fi.

---

## Troubleshooting

**`flutter devices` shows nothing**
- Android: check USB debugging is enabled; try a different cable
- iOS: make sure you tapped Trust on the phone and Xcode has registered the device

**App connects but can't reach butler**
- Confirm butler is running: `curl http://<butler-ip>:8080/api/v1/auth/status`
- Check firewall isn't blocking port 8080

**iOS 7-day expiry**
A free Apple developer account signs apps valid for 7 days. After that, re-run `flutter run` to re-sign and reinstall. A paid developer account ($99/yr) gets you longer-lived distribution profiles.
