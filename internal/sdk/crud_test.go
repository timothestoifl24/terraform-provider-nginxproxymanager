// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package go_npm_proxy_manager

// This file only contains tests for the CRUD utilities, not
// the main CRUD functions. Those tests are in the test files
// for each object type.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/edklesel/proto"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
)

func compareObjects(t *testing.T, obj1 any, obj2 any, _ignores []string) {

	var final = []string{}
	var ignores = []*regexp.Regexp{}
	var match bool

	for _, ignore := range _ignores {
		ignores = append(ignores, regexp.MustCompile(ignore))
	}

	for _, msg := range deep.Equal(obj1, obj2) {
		match = false
		for _, ignore := range ignores {

			if match = ignore.Match([]byte(msg)); match {
				break
			}
		}
		if !match {
			final = append(final, msg)
		}
	}
	assert.Empty(t, final)
}

type withBaseProperties interface {
	Object | ObjectCreate
}

func getBaseProperties[O withBaseProperties, OB ObjectBase](object O) (objectBase OB) {

	objectName := reflect.ValueOf(object).Type().Name()

	var objectBaseName string

	// If we're looking at the Create object, string the Create and add Base
	if strings.Contains(objectName, "Create") {
		objectBaseName = strings.Replace(objectName, "Create", "Base", 1)
		// Otherwise if it's the Base object (which most Create objects are) then
		// just use this nase
	} else if objectName == reflect.ValueOf(objectBase).Type().Name() {
		objectBaseName = reflect.ValueOf(objectBase).Type().Name()
		// Otherwise append Base to the end
	} else {
		objectBaseName = fmt.Sprintf("%sBase", objectName)
	}

	var _base []byte
	// If the Object isn't the Base object, extract the base object properties
	if objectName != objectBaseName {
		_objectBase := reflect.ValueOf(object).FieldByName(objectBaseName).Interface()
		_base, _ = json.Marshal(_objectBase)

	} else { // Otherwise use the object
		_base, _ = json.Marshal(object)
	}

	// Marshal it into an OB type variable
	_ = json.Unmarshal(_base, &objectBase)

	return objectBase
}

func testCRD[O Object, OB ObjectBase, OC ObjectCreate](t *testing.T, c CRD[O, OB, OC]) {

	var obj O
	objectName := reflect.ValueOf(obj).Type().Name()

	t.Run(fmt.Sprintf("Create %s", objectName), func(t *testing.T) {

		// Create a prototype object
		var objectCreate OC
		proto.PrototypeWithVersion(&objectCreate, c.ClientVersion())

		object, err := c.Create(objectCreate)
		assert.NoError(t, err)

		// Pull out the ID and base properties of the object and the create object
		objectId := getID(object)
		assert.NotZero(t, objectId)

		objectBase := getBaseProperties[O, OB](object)
		objectCreateBase := getBaseProperties[OC, OB](objectCreate)

		compareObjects(t, objectCreateBase, objectBase, c.GetWriteOnlyFields())

		// Get the object from the DB and make sure the properties match
		objectFetch, err := c.Get(objectId)
		assert.NoError(t, err)
		objectFetchBase := getBaseProperties[O, OB](objectFetch)

		compareObjects(t, objectBase, objectFetchBase, c.GetWriteOnlyFields())

		// Tidy up
		err = c.Delete(objectId)
		if err != nil {
			t.Errorf("Unable to delete %s with ID %d: %v", objectName, objectId, err)
		}

	})

	t.Run(fmt.Sprintf("List %s objects", objectName), func(t *testing.T) {

		var objectCreate OC
		proto.PrototypeWithVersion(&objectCreate, c.ClientVersion())

		object, err := c.Create(objectCreate)
		if err != nil {
			t.Errorf("Failed to create %s object: %v", objectName, err)
		}
		objectId := getID(object)
		assert.NoError(t, err)

		var object2Create OC
		proto.PrototypeWithVersion(&object2Create, c.ClientVersion())

		object2, err := c.Create(object2Create)
		object2Id := getID(object2)
		assert.NoError(t, err)

		var objects []O
		objects, err = c.List()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(objects), 2)

		// Tidy up
		for _, i := range []int{objectId, object2Id} {
			err = c.Delete(i)
			if err != nil {
				t.Errorf("Unable to delete %s with ID %d: %v", objectName, i, err)
			}
		}
	})

	t.Run(fmt.Sprintf("Check %s can be retrieved", objectName), func(t *testing.T) {

		var objectCreate OC
		proto.PrototypeWithVersion(&objectCreate, c.ClientVersion())

		object, err := c.Create(objectCreate)
		objectId := getID(object)
		assert.NoError(t, err)

		var objectGet O
		objectGet, err = c.Get(objectId)
		assert.NoError(t, err)

		diffs := deep.Equal(object, objectGet)
		assert.Empty(t, diffs)

		// Tidy up
		err = c.Delete(objectId)
		if err != nil {
			t.Errorf("Unable to delete %s with ID %d: %v", objectName, objectId, err)
		}
	})

	t.Run(fmt.Sprintf("Check non-existent %s is not found", objectName), func(t *testing.T) {

		var objectCreate OC
		proto.PrototypeWithVersion(&objectCreate, c.ClientVersion())

		object, err := c.Create(objectCreate)
		assert.NoError(t, err)
		objectId := getID(object)

		nonID := objectId + 1e4

		_, err = c.Get(nonID)
		assert.True(t, IsNotFound(err), "expected a not found error, got %v", err)
		assert.ErrorContains(t, err, fmt.Sprintf("%s with id %d not found", strings.ToLower(objectName), nonID))

		// Tidy up
		err = c.Delete(objectId)
		if err != nil {
			t.Errorf("Unable to delete %s with ID %d: %v", objectName, objectId, err)
		}
	})

	t.Run(fmt.Sprintf("Delete %s", objectName), func(t *testing.T) {

		var objectCreate OC
		proto.PrototypeWithVersion(&objectCreate, c.ClientVersion())

		object, err := c.Create(objectCreate)
		assert.NoError(t, err)
		objectId := getID(object)

		// Try deleting it
		err = c.Delete(objectId)
		assert.NoError(t, err)

		// Ensure it now can't be found
		_, err = c.Get(objectId)
		assert.True(t, IsNotFound(err), "expected a not found error, got %v", err)
		assert.ErrorContains(t, err, fmt.Sprintf("%s with id %d not found", strings.ToLower(objectName), objectId))

		// Ensure it can't be deleted again
		err = c.Delete(objectId)
		assert.True(t, IsNotFound(err), "expected a not found error, got %v", err)

	})

}

func testCRUD[O Object, OB ObjectBase, OC ObjectCreate](t *testing.T, c CRUD[O, OB, OC]) {

	var obj O
	objectName := reflect.ValueOf(obj).Type().Name()

	testCRD[O, OB](t, c)

	t.Run(fmt.Sprintf("Update %s", objectName), func(t *testing.T) {

		var objectCreate OC
		proto.PrototypeWithVersion(&objectCreate, c.ClientVersion())

		object, err := c.Create(objectCreate)
		assert.NoError(t, err)
		objectId := getID(object)

		// Modify the user properties in a cloned struct
		_object := object
		var objectUpdate = getBaseProperties[O, OB](_object)
		proto.ModifyWithVersion(&objectUpdate, c.ClientVersion())

		// Update the object in the API
		var objectPostUpdate O
		objectPostUpdate, err = c.Update(objectId, objectUpdate)
		assert.NoError(t, err)

		// Pull out the object base properties
		objectPostUpdateBase := getBaseProperties[O, OB](objectPostUpdate)

		assert.Nil(t, deep.Equal(objectPostUpdateBase, objectUpdate))

		// Tidy up
		err = c.Delete(objectId)
		if err != nil {
			t.Errorf("Unable to delete %s with ID %d: %v", objectName, objectId, err)
		}
	})
}
