package main

import "io/ioutil"
import "os"
import "testing"

/* The test devices we're going to use.
 * The values for Latitude, Longitude and Timestamp are the ones
 * we expect to find right after adding them to the database for the
 * first time without updating their location.
 */
var gTestDevices = []Device{
	{Id: 1, User: "ggp@mozilla.com", Name: "test-device1",
		Endpoint: "http://push.mozilla.com/83c8e238-be79-41de-9782-b9ce207d0ec1",
		Latitude: 0, Longitude: 0, Timestamp: ""},

	{Id: 2, User: "ggp@mozilla.com", Name: "test-device2",
		Endpoint: "http://push.mozilla.com/1e16a9e1-b5c7-4d79-86c0-0724117b2fde",
		Latitude: 0, Longitude: 0, Timestamp: ""},

	{Id: 3, User: "ggoncalves@mozilla.com", Name: "test-device3",
		Endpoint: "http://push.mozilla.com/f8303f58-f486-4ed7-8dd7-3a741837ff51",
		Latitude: 0, Longitude: 0, Timestamp: ""},
}

var gTestCommands = []Command{
	{Id: 1, Name: "Track", Description: "Start tracking a device"},
	{Id: 2, Name: "Untrack", Description: "Stop tracking a device"},
	{Id: 3, Name: "Wipe", Description: "Wipe a device's personal information"},
}

func initTestDatabase(t *testing.T) (*DB, func()) {
	testDBFile, err := ioutil.TempFile("", "whereismyfoxdb")
	if err != nil {
		panic(err)
	}

	testDBPath := testDBFile.Name()
	db, err := OpenDB(testDBPath)

	for _, command := range gTestCommands {
		_, err := db.AddCommand(command.Id, command.Name, command.Description)
		if err != nil {
			t.Log("Failed to add command: " + err.Error())
			t.FailNow()
		}
	}

	for i, device := range gTestDevices {
		added, err := db.AddDevice(device.User, device.Name, device.Endpoint)

		if err != nil {
			t.Log("Failed to add device: " + err.Error())
			t.FailNow()
		}

		if *added != device {
			t.Errorf("Mismatch in added device: %#v != %#v", *added, device)
		}

		for j, command := range gTestCommands {
			if j > i {
				break
			}

			err = db.AddCommandForDevice(added.Id, command.Id)
			if err != nil {
				t.Log("Failed to add command for device: " + err.Error())
				t.FailNow()
			}
		}
	}

	return db, func() {
		db.Close()
		os.Remove(testDBPath)
	}
}

func TestGetDeviceById(t *testing.T) {
	db, cleanup := initTestDatabase(t)
	defer cleanup()

	// First try to retrieve a known device
	device, err := db.GetDeviceById(1)
	if err != nil {
		t.Error("Failed to get device by id: " + err.Error())
	}

	if device.Id != 1 {
		t.Errorf("Unexpected device id %d (expected 1)", device.Id)
	}

	// Now try an inexistent device
	device, err = db.GetDeviceById(42)
	if device != nil {
		t.Errorf("Found a device with inexistent id: %#v", device)
	}
}

func TestUpdateDeviceLocation(t *testing.T) {
	db, cleanup := initTestDatabase(t)
	defer cleanup()

	testLatitude := 37.38835
	testLongitude := -122.082724

	if device, err := db.GetDeviceById(1); err == nil {
		err = db.UpdateDeviceLocation(device, testLatitude, testLongitude)
		if err != nil {
			t.Error("Failed to update device location: " + err.Error())
		}
	} else {
		panic(err)
	}

	if device, err := db.GetDeviceById(1); device != nil {
		if device.Latitude != testLatitude || device.Longitude != testLongitude {
			t.Errorf("Device has wrong coordinates: %#v", device)
		}

		if device.Timestamp == "" {
			t.Errorf("Timestamp for device was not updated: %#v", device)
		}
	} else {
		panic(err)
	}
}

func TestListDevicesForUser(t *testing.T) {
	db, cleanup := initTestDatabase(t)
	defer cleanup()

	devices, err := db.ListDevicesForUser("ggp@mozilla.com")
	if err != nil {
		t.Error("Failed to list devices: " + err.Error())
	}

	if len(devices) != 2 {
		t.Errorf("Found wrong number of devices: %d", len(devices))
	}

	for _, device := range devices {
		if device.User != "ggp@mozilla.com" {
			t.Errorf("Wrong user: %s", devices[0].User)
		}
	}
}

func TestListCommandsForDevice(t *testing.T) {
	db, cleanup := initTestDatabase(t)
	defer cleanup()

	for i, device := range gTestDevices {
		commands, err := db.ListCommandsForDevice(&device)

		if err != nil {
			t.Error("Failed to list commands: " + err.Error())
		}

		if len(commands) != i+1 {
			t.Errorf("Wrong number of commands for device %d: %d", device.Id,
				len(commands))
		}
	}
}

func TestUpdateCommandsForDevice(t *testing.T) {
	db, cleanup := initTestDatabase(t)
	defer cleanup()

	device, _ := db.GetDeviceById(1)
	newCommands := []int64{1, 2, 3}

	if err := db.UpdateCommandsForDevice(device.Id, newCommands); err != nil {
		t.Errorf("Failed to update commands for device: %s", err.Error())
	}

	commands, _ := db.ListCommandsForDevice(device)
	if len(commands) != len(newCommands) {
		t.Errorf("Unexpected number of commands: %d", len(commands))
	}
}
