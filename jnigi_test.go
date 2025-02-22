// Copyright 2016 Tim O'Brien. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jnigi

import (
	"testing"
)

var env *Env
var jvm *JVM


// Run them all here so we can be sure they run on the same Goroutine
func TestAll(t *testing.T) {
	PTestInit(t)
	PTestBasic(t)
	PTestObjectArrays(t)
	PTestInstanceOf(t)
	PTestByteArray(t)
	PTestAttach(t)
	PTestMonitor(t)
	PTestGetJVM(t)
	PTestEnsureLocalCapacity(t)
	PTestPushPopLocalFrame(t)
	PTestHandleException(t)
	PTestDestroy(t)
}

func PTestInit(t *testing.T) {
	libPath := AttemptToFindJVMLibPath()
	if err := LoadJVMLib(libPath); err != nil {
		t.Logf("library path = %s", libPath)
		t.Log("can use JAVA_HOME environment variable to set JRE root directory")
		t.Fatal(err)
	}
	jvm2, e2, err := CreateJVM(NewJVMInitArgs(false, true, DEFAULT_VERSION, []string{"-Xcheck:jni"}))
	if err != nil {
		t.Fatal(err)
	}
	env = e2
	jvm = jvm2

	t.Logf("%x", e2.jniEnv)

}

func toGoStr(t *testing.T, o *ObjectRef) string {
	v, err := o.CallMethod(env, "getBytes", Byte|Array)
	if err != nil {
		t.Fatal(err)
	}
	return string(v.([]byte))
}

func fromGoStr(t *testing.T, str string) *ObjectRef {
	jstr, err := env.NewObject("java/lang/String", []byte(str))
	if err != nil {
		t.Fatal(err)
	}
	return jstr
}

func PTestBasic(t *testing.T) {
	// new object, int method
	obj, err := env.NewObject("java/lang/Object")
	if err != nil {
		t.Fatal(err)
	}
	v, err := obj.CallMethod(env, "hashCode", Int)
	if err != nil {
		t.Fatal(err)
	}

	// byte array argument, byte array method
	testStr := "hello world"
	str, err := env.NewObject("java/lang/String", []byte(testStr))
	if err != nil {
		t.Fatal(err)
	}
	v, err = str.CallMethod(env, "getBytes", Byte|Array, env.GetUTF8String())
	if err != nil {
		t.Fatal(err)
	}
	if b, ok := v.([]byte); !ok || string(b) != testStr {
		t.Logf("basic test failed")
	}

	// object method, int arg, object arg
	v, err = str.CallMethod(env, "substring", "java/lang/String", 6)
	if err != nil {
		t.Fatal(err)
	}
	str2 := v.(*ObjectRef)
	v, err = str2.CallMethod(env, "getBytes", Byte|Array)
	if err != nil {
		t.Fatal(err)
	}

	env.PrecalculateSignature("(Ljava/lang/String;)Z")
	v, err = str.CallMethod(env, "endsWith", Boolean, str2)
	if err != nil {
		t.Fatal(err)
	}
	if b, ok := v.(bool); !ok || !b {
		t.Logf("basic test failed")
	}

	// call static method
	v, err = env.CallStaticMethod("java/lang/System", "getProperty", "java/lang/String", fromGoStr(t, "java.vm.version"))
	t.Logf(toGoStr(t, v.(*ObjectRef)))

	// get static field
	v, err = env.GetStaticField("java/util/Calendar", "APRIL", Int)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("april = %d", v.(int))

	// set/get object field
	pt, err := env.NewObject("java/awt/Point")
	if err != nil {
		t.Fatal(err)
	}
	err = pt.SetField(env, "x", 5)
	if err != nil {
		t.Fatal(err)
	}
	v, err = pt.GetField(env, "x", Int)
	if err != nil {
		t.Fatal(err)
	}
	if i, ok := v.(int); !ok || i != 5 {
		t.Logf("basic test failed")
	}
	
	src := "fromChar"
	dest := make([]uint16, len(src))
	for i, c := range src {
		dest[i] = uint16(c)
	}
	str, err = env.NewObject("java/lang/String", dest)
	if err != nil {
		t.Fatal(err)
	}

	v, err = str.CallMethod(env, "getBytes", Byte|Array, env.GetUTF8String())
	if err != nil {
		t.Fatal(err)
	}
	if b, ok := v.([]byte); !ok || string(b) != src {
		t.Logf("basic test failed")
	}
	
}

func PTestAttach(t *testing.T) {
	x := make(chan byte)

	obj, err := env.NewObject("java/lang/Object")
	if err != nil {
		t.Fatal(err)
	}
	gObj := env.NewGlobalRef(obj)

	go func() {
		nenv := jvm.AttachCurrentThread()
		t.Logf("%x", nenv.jniEnv)

		v, err := gObj.CallMethod(nenv, "hashCode", Int)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%d", v.(int))
		if err := jvm.DetachCurrentThread(); err != nil {
			t.Fatal(err)
		}

		x <- 4
	}()

	<-x
}

func PTestMonitor(t *testing.T) {
	obj, err := env.NewObject("java/lang/Object")
	if err != nil {
		t.Fatal(err)
	}
	x := env.MonitorEnter(obj)
	y := env.MonitorExit(obj)
	if !x || !y {
		t.Fatalf("one of both monitor method was return negative value")
	}
}

func PTestObjectArrays(t *testing.T) {
	str, err := env.NewObject("java/lang/String", []byte("splitXme"))
	if err != nil {
		t.Fatal(err)
	}

	regex, err := env.NewObject("java/lang/String", []byte("X"))
	if err != nil {
		t.Fatal(err)
	}

	v, err := str.CallMethod(env, "split", ObjectArrayType("java/lang/String"), regex)
	if err != nil {
		t.Fatal(err)
	}

	parts := env.FromObjectArray(v.(*ObjectRef))
	for _, p := range parts {
		v, err = p.CallMethod(env, "getBytes", Byte|Array)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s", string(v.([]byte)))
	}

	array := env.ToObjectArray(parts, "java/lang/String")

	v, err = array.CallMethod(env, "getClass", "java/lang/Class")
	if err != nil {
		t.Fatal(err)
	}
	v, err = v.(*ObjectRef).CallMethod(env, "getName", "java/lang/String")
	if err != nil {
		t.Fatal(err)
	}
	v, err = v.(*ObjectRef).CallMethod(env, "getBytes", Byte|Array)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", v.([]byte))
}

func PTestInstanceOf(t *testing.T) {
	alist, err := env.NewObject("java/util/ArrayList")
	if err != nil {
		t.Fatal(err)
	}

	str, err := env.NewObject("java/lang/String")
	if err != nil {
		t.Fatal(err)
	}
	_, err = alist.CallMethod(env, "add", Boolean, str.Cast("java/lang/Object"))
	if err != nil {
		t.Fatal(err)
	}

	v, err := alist.CallMethod(env, "get", "java/lang/Object", 0)
	if err != nil {
		t.Fatal(err)
	}
	obj := v.(*ObjectRef)

	if v, err := obj.IsInstanceOf(env, "java/lang/String"); err != nil {
		t.Fatal(err)
	} else if !v {
		t.Fatal("instanceof test failed")
	}
}

func PTestByteArray(t *testing.T) {
	ba := env.NewByteArray(5)
	bytes := ba.GetCritical(env)
	copy(bytes, []byte("hello"))
	ba.ReleaseCritical(env, bytes)
	str, err := env.NewObject("java/lang/String", ba)
	if err != nil {
		t.Fatal(err)
	}
	if toGoStr(t, str) != "hello" {
		t.Fatal("ByteArray test failed")
	}

	testStr := "hello world"
	str, err = env.NewObject("java/lang/String", []byte(testStr))
	if err != nil {
		t.Fatal(err)
	}
	
	env.NoReturnConvert()
	v, err := str.CallMethod(env, "getBytes", Byte|Array, env.GetUTF8String())
	if err != nil {
		t.Fatal(err)
	}

	o, ok := v.(*ObjectRef)
	if !ok {
		t.Logf("ByteArray test failed")
	}

	ba2 := env.NewByteArrayFromObject(o)
	bytes = ba2.GetCritical(env)
	if string(bytes) != "hello world" {
		t.Logf("ByteArray test failed")
	}
	ba2.ReleaseCritical(env, bytes)
}

func PTestGetJVM(t *testing.T) {
	_, err := env.GetJVM()
	if err != nil {
		t.Fatalf("GetJavaVM failed %s", err)
	}
	t.Logf("Call GetJavaJVM: passed")
}

func PTestDestroy(t *testing.T) {
	err := jvm.Destroy()
	if err != nil {
		t.Fatalf("DestroyJVM failed %s", err)
	}
	t.Logf("Call DestroyJVM: passed")
}

func PTestEnsureLocalCapacity(t *testing.T) {
	if err := env.EnsureLocalCapacity(256); err != nil {
		t.Fatalf("EnsureLocalCapacity failed %s", err)
	}
	t.Logf("Call EnsureLocalCapacity: passed")
}

func PTestPushPopLocalFrame(t *testing.T) {
	if err := env.PushLocalFrame(64); err != nil {
		t.Fatalf("PushLocalFrame failed %s", err)
	}
	t.Logf("Call PushLocalFrame: passed")

	obj, err := env.NewObject("java/lang/Object")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := obj.CallMethod(env, "hashCode", Int); err != nil {
		t.Fatal(err)
	}

	// Pop local frame with obj reference; obj should now be in previous frame
	obj = env.PopLocalFrame(obj)
	t.Logf("Call PopLocalFrame: passed")

	if _, err := obj.CallMethod(env, "hashCode", Int); err != nil {
		t.Fatalf("hashCode after PopLocalFrame failed %s", err)
	}

	env.DeleteLocalRef(obj)

	// Now do again with nil argument to pop
	if err := env.PushLocalFrame(32); err != nil {
		t.Fatalf("PushLocalFrame failed %s", err)
	}
	t.Logf("Call PushLocalFrame: passed")

	obj, err = env.NewObject("java/lang/Object")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := obj.CallMethod(env, "hashCode", Int); err != nil {
		t.Fatal(err)
	}

	// Pop local frame with nil
	obj = env.PopLocalFrame(nil)
	t.Logf("Call PopLocalFrame: passed")

	if !obj.IsNil() {
		t.Fatal("PopLocalFrame return value is not nil")
	}
}

func PTestHandleException(t *testing.T) {

	if _, err := env.NewObject("java/foo/bar"); err == nil {
		t.Fatal("did not return error")
	} else if err.Error() != "Java exception occured. check stderr" {
		t.Fatalf("did not return standard error: %v", err)
	}

	env.ExceptionHandler = ThrowableToStringExceptionHandler
	if _, err := env.NewObject("java/foo/bar"); err == nil {
		t.Fatal("did not return error")
	} else if err.Error() != "java.lang.NoClassDefFoundError: java/foo/bar" {
		t.Fatalf("unexpected result of ToString: %v", err)
	}

	env.ExceptionHandler = ThrowableErrorExceptionHandler
	if _, err := env.NewObject("java/foo/bar"); err == nil {
		t.Fatal("did not return error")
	} else {

		throwableError, ok := err.(ThrowableError)
		if !ok {
			t.Fatalf("expected ThrowableError, but got %T", err)
		}

		if err.Error() != "java.lang.NoClassDefFoundError: java/foo/bar" {
			t.Fatalf("unexpected error message: %v", err)
		}

		if v := throwableError.ClassName; v != "java.lang.NoClassDefFoundError" {
			t.Fatalf("unexpected class name: %s", v)
		}

		if v := throwableError.LocalizedMessage; v != "java/foo/bar" {
			t.Fatalf("unexpected localized message: %s", v)
		}

		if v := throwableError.Message; v != "java/foo/bar" {
			t.Fatalf("unexpected message: %s", v)
		}

		if v := throwableError.AsString; v != "java.lang.NoClassDefFoundError: java/foo/bar" {
			t.Fatalf("unexpected toString value: %s", v)
		}

		if v := throwableError.StackTrace; len(v) > 0 {
			t.Fatal("expect empty stack trace")
		}

		if throwableError.Cause == nil {
			t.Fatal("expected a cause")
		}

		cause := throwableError.Cause

		if cause.Error() != "java.lang.ClassNotFoundException: java.foo.bar" {
			t.Fatalf("unexpected error message: %v", cause)
		}

		if v := cause.ClassName; v != "java.lang.ClassNotFoundException" {
			t.Fatalf("unexpected class name: %s", v)
		}

		if v := cause.LocalizedMessage; v != "java.foo.bar" {
			t.Fatalf("unexpected localized message: %s", v)
		}

		if v := cause.Message; v != "java.foo.bar" {
			t.Fatalf("unexpected message: %s", v)
		}

		if v := cause.AsString; v != "java.lang.ClassNotFoundException: java.foo.bar" {
			t.Fatalf("unexpected toString value: %s", v)
		}

		if v := cause.StackTrace; v == nil {
			t.Fatal("expected a stack trace")
		} else if len(cause.StackTrace) == 0 {
			t.Fatal("expected stack trace entries")
		}

		for i, v := range cause.StackTrace {
			if v.AsString == "" {
				t.Fatalf("stack trace index %d: no AsString value", i)
			}
			if v.ClassName == "" {
				t.Fatalf("stack trace index %d: no ClassName value", i)
			}
			if v.FileName == "" {
				t.Fatalf("stack trace index %d: no FileName value", i)
			}
			if v.MethodName == "" {
				t.Fatalf("stack trace index %d: no MethodName value", i)
			}
			if v.LineNumber == 0 {
				t.Fatalf("stack trace index %d: no LineNumber value", i)
			}
		}
	}

	env.ExceptionHandler = nil

	if _, err := env.NewObject("java/foo/bar"); err == nil {
		t.Fatal("did not return error")
	} else if err.Error() != "Java exception occured. check stderr" {
		t.Fatalf("did not return standard error: %v", err)
	}
}
