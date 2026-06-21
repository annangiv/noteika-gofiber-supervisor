/** @type {import('@capacitor/cli').CapacitorConfig} */

// Remote-server mode: the native shell loads your deployed Noteika web app.
// Override for local dev:
//   NOTEIKA_SERVER_URL=http://10.0.2.2:8080 npm run cap:sync   (Android emulator)
//   NOTEIKA_SERVER_URL=http://localhost:8080 npm run cap:sync    (iOS simulator)
const serverUrl = process.env.NOTEIKA_SERVER_URL || 'http://localhost:8080';

module.exports = {
  appId: 'app.noteika.mobile',
  appName: 'Noteika',
  webDir: '../static',
  server: {
    url: serverUrl,
    cleartext: serverUrl.startsWith('http://'),
    androidScheme: serverUrl.startsWith('https://') ? 'https' : 'http',
  },
  ios: {
    contentInset: 'automatic',
  },
  android: {
    allowMixedContent: true,
  },
  plugins: {
    SplashScreen: {
      launchAutoHide: true,
      backgroundColor: '#0d1117',
      showSpinner: false,
    },
    StatusBar: {
      style: 'DARK',
      backgroundColor: '#0d1117',
    },
  },
};
