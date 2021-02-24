#!/bin/bash
set -e

function test() {
    adb -s alice:5555 shell am instrument -e log false -w -e debug false -e class "network.perun.app.PrnmTest#run$1TestAlice" network.perun.app.test/androidx.test.runner.AndroidJUnitRunner 2>&1 &
    ALICE=$!
    adb -s bob:5555 shell am instrument -e log false -w -e debug false -e class "network.perun.app.PrnmTest#run$1TestBob" network.perun.app.test/androidx.test.runner.AndroidJUnitRunner 2>&1 &
    BOB=$!
    wait $ALICE || exit 1
    wait $BOB || exit 1
}

echo 'Copy'
cp -R /src/ /pwd/
cd /pwd/
echo 'prnm: Generating bindings, this takes some time'
gomobile bind -o android/app/prnm.aar -target=android
echo 'prnm: Connecting to emulators'
cd android
rm -rf .gradle app/build/
gradle -w clean
adb connect alice:5555
adb connect bob:5555
adb devices
echo 'prnm: Compiling tests'
gradle -w installDebugAndroidTest installDebug
echo 'prnm: Setting up networking'
adb -s   bob:5555 forward tcp:5751 tcp:5750
adb -s alice:5555 forward tcp:5752 tcp:5750
(socat tcp4-listen:5750,reuseaddr,fork tcp:localhost:5751 &)
(socat tcp4-listen:5753,reuseaddr,fork tcp:localhost:5752 &)
(adb -s alice:5555 logcat -b main &)
(adb -s bob:5555   logcat -b main &)

echo 'prnm: Starting first tests'
test "First"
echo 'prnm: Starting second tests'
test "Second"
