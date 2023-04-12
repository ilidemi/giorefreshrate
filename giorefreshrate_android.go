//go:build android

package giorefreshrate

/*
#cgo CFLAGS: -Werror
#cgo LDFLAGS: -landroid

#include <android/native_window_jni.h>
#include <android/configuration.h>
#include <android/keycodes.h>
#include <android/input.h>
#include <stdlib.h>

static jint jni_GetEnv(JavaVM *vm, JNIEnv **env, jint version) {
	return (*vm)->GetEnv(vm, (void **)env, version);
}

static jint jni_AttachCurrentThread(JavaVM *vm, JNIEnv **p_env, void *thr_args) {
	return (*vm)->AttachCurrentThread(vm, p_env, thr_args);
}

static jint jni_DetachCurrentThread(JavaVM *vm) {
	return (*vm)->DetachCurrentThread(vm);
}

static jclass jni_GetObjectClass(JNIEnv *env, jobject obj) {
	return (*env)->GetObjectClass(env, obj);
}

static jmethodID jni_GetMethodID(JNIEnv *env, jclass clazz, const char *name, const char *sig) {
	return (*env)->GetMethodID(env, clazz, name, sig);
}

static jfieldID jni_GetFieldID(JNIEnv *env, jclass clazz, const char *name, const char *sig) {
    return (*env)->GetFieldID(env, clazz, name, sig);
}

static jint jni_CallIntMethodA(JNIEnv *env, jobject obj, jmethodID methodID, jvalue *args) {
	return (*env)->CallIntMethodA(env, obj, methodID, args);
}

static jfloat jni_CallFloatMethodA(JNIEnv *env, jobject obj, jmethodID methodID, jvalue *args) {
	return (*env)->CallFloatMethodA(env, obj, methodID, args);
}

static void jni_CallVoidMethodA(JNIEnv *env, jobject obj, jmethodID methodID, const jvalue *args) {
	(*env)->CallVoidMethodA(env, obj, methodID, args);
}

static jsize jni_GetArrayLength(JNIEnv *env, jarray arr) {
	return (*env)->GetArrayLength(env, arr);
}

static jobject jni_GetObjectArrayElement(JNIEnv *env, jobjectArray arr, jsize index) {
    return (*env)->GetObjectArrayElement(env, arr, index);
}

static jstring jni_NewString(JNIEnv *env, const jchar *unicodeChars, jsize len) {
	return (*env)->NewString(env, unicodeChars, len);
}

static jsize jni_GetStringLength(JNIEnv *env, jstring str) {
	return (*env)->GetStringLength(env, str);
}

static const jchar *jni_GetStringChars(JNIEnv *env, jstring str) {
	return (*env)->GetStringChars(env, str, NULL);
}

static jthrowable jni_ExceptionOccurred(JNIEnv *env) {
	return (*env)->ExceptionOccurred(env);
}

static void jni_ExceptionClear(JNIEnv *env) {
	(*env)->ExceptionClear(env);
}

static jobject jni_CallObjectMethodA(JNIEnv *env, jobject obj, jmethodID method, jvalue *args) {
	return (*env)->CallObjectMethodA(env, obj, method, args);
}

static jclass jni_FindClass(JNIEnv *env, char *name) {
	return (*env)->FindClass(env, name);
}

static void jni_SetIntField(JNIEnv *env, jobject obj, jfieldID field, jint value) {
	return (*env)->SetIntField(env, obj, field, value);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"unicode/utf16"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/system"
)

var viewPtr uintptr

func listenEvents(event event.Event, w *app.Window) error {
	switch event := event.(type) {
	case app.ViewEvent:
		viewPtr = event.View
	case system.StageEvent:
		if event.Stage == system.StageRunning {
			if preference == refreshRateNone {
				return errors.New(
					"it's time to set the refresh rate but no preference was specified. Call PreferHighRefreshRate() or PreferLowRefreshRate() before starting the event loop",
				)
			}

			if viewPtr == 0 {
				return errors.New("received StageEvent without a preceding ViewEvent")
			}

			setRefreshRate(w, preference)
		}
	}

	return nil
}

// Sets the refresh rate. Doesn't propagate errors as the call loops through the main thread.
func setRefreshRate(w *app.Window, preference refreshRatePreference) {
	w.Run(func() {
		err := runInJVM(javaVM(), func(env *C.JNIEnv) error {
			// context = surfaceView.getContext();
			surfaceView := C.jobject(viewPtr)
			surfaceViewClass := getObjectClass(env, surfaceView)
			getContext := getMethodID(env, surfaceViewClass, "getContext", "()Landroid/content/Context;")
			context, err := callObjectMethod(env, surfaceView, getContext)
			if err != nil {
				return err
			}

			// display = context.getDisplay();
			contextClass := getObjectClass(env, context)
			getDisplay := getMethodID(env, contextClass, "getDisplay", "()Landroid/view/Display;")
			display, err := callObjectMethod(env, context, getDisplay)
			if err != nil {
				return err
			}

			// currentMode = display.getMode();
			displayClass := getObjectClass(env, display)
			getMode := getMethodID(env, displayClass, "getMode", "()Landroid/view/Display$Mode;")
			currentMode, err := callObjectMethod(env, display, getMode)
			if err != nil {
				return err
			}

			// currentWidth = currentMode.getPhysicalWidth();
			modeClass := getObjectClass(env, currentMode)
			getPhysicalWidth := getMethodID(env, modeClass, "getPhysicalWidth", "()I")
			currentWidth, err := callIntMethod(env, currentMode, getPhysicalWidth)
			if err != nil {
				return err
			}

			// currentHeight = currentMode.getPhysicalHeight();
			getPhysicalHeight := getMethodID(env, modeClass, "getPhysicalHeight", "()I")
			currentHeight, err := callIntMethod(env, currentMode, getPhysicalHeight)
			if err != nil {
				return err
			}

			// supportedModes = display.getSupportedModes();
			getSupportedModes := getMethodID(
				env, displayClass, "getSupportedModes", "()[Landroid/view/Display$Mode;",
			)
			supportedModes, err := callObjectMethod(env, display, getSupportedModes)
			if err != nil {
				return err
			}

			// Iterate over supported modes
			length := getObjectArrayLength(env, C.jobjectArray(supportedModes))
			getRefreshRate := getMethodID(env, modeClass, "getRefreshRate", "()F")
			getModeId := getMethodID(env, modeClass, "getModeId", "()I")
			var bestRefreshRate float32
			var bestModeId int32
			for i := 0; i < length; i++ {
				// mode = supportedModes[i];
				mode, err := getObjectArrayElement(env, C.jobjectArray(supportedModes), C.jsize(i))
				if err != nil {
					return err
				}

				// refreshRate = mode.getRefreshRate();
				refreshRate, err := callFloatMethod(env, mode, getRefreshRate)
				if err != nil {
					return err
				}

				// width = mode.getPhysicalWidth();
				width, err := callIntMethod(env, mode, getPhysicalWidth)
				if err != nil {
					return err
				}

				// height = mode.getPhysicalHeight();
				height, err := callIntMethod(env, mode, getPhysicalHeight)
				if err != nil {
					return err
				}

				refreshRateIsBetter := bestRefreshRate == 0 ||
					(preference == refreshRateHigh && refreshRate > bestRefreshRate) ||
					(preference == refreshRateLow && refreshRate < bestRefreshRate)

				if width == currentWidth && height == currentHeight && refreshRateIsBetter {
					// modeId = mode.getModeId();
					modeId, err := callIntMethod(env, mode, getModeId)
					if err != nil {
						return err
					}

					bestRefreshRate = refreshRate
					bestModeId = modeId
				}
			}

			if bestRefreshRate == 0 {
				return errors.New("none of the available display modes are matching the current resolution")
			}

			// activity = (Activity)context;
			activityClass := findClass(env, "android/app/Activity")
			activity := context

			// window = activity.getWindow();
			getWindow := getMethodID(env, activityClass, "getWindow", "()Landroid/view/Window;")
			window, err := callObjectMethod(env, activity, getWindow)
			if err != nil {
				return err
			}

			// layoutParams = window.getAttributes();
			windowClass := getObjectClass(env, window)
			getAttributes := getMethodID(
				env, windowClass, "getAttributes", "()Landroid/view/WindowManager$LayoutParams;",
			)
			layoutParams, err := callObjectMethod(env, window, getAttributes)
			if err != nil {
				return err
			}

			// layoutParams.preferredDisplayModeId = bestModeId;
			layoutParamsClass := getObjectClass(env, layoutParams)
			preferredDisplayModeIdField := getFieldID(env, layoutParamsClass, "preferredDisplayModeId", "I")
			setIntField(env, layoutParams, preferredDisplayModeIdField, bestModeId)

			// This is the call that needs to happen on the main thread
			// window.setAttributes(layoutParams);
			setAttributes := getMethodID(
				env, windowClass, "setAttributes", "(Landroid/view/WindowManager$LayoutParams;)V",
			)
			err = callVoidMethod(env, window, setAttributes, jvalue(layoutParams))
			if err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			fmt.Println("set refresh rate:", err)
		}
	})
}

// JNI interop
// git.wow.st/gmp/jni is not used here because it doesn't provide GetObjectArrayLength() and SetIntField()

type jvalue uint64 // The largest JNI type fits in 64 bits.

func runInJVM(jvm *C.JavaVM, f func(env *C.JNIEnv) error) error {
	if jvm == nil {
		panic("nil JVM")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var env *C.JNIEnv
	if res := C.jni_GetEnv(jvm, &env, C.JNI_VERSION_1_6); res != C.JNI_OK {
		if res != C.JNI_EDETACHED {
			panic(fmt.Errorf("JNI GetEnv failed with error %d", res))
		}
		if C.jni_AttachCurrentThread(jvm, &env, nil) != C.JNI_OK {
			panic(errors.New("runInJVM: AttachCurrentThread failed"))
		}
		defer C.jni_DetachCurrentThread(jvm)
	}

	return f(env)
}

func javaVM() *C.JavaVM {
	return (*C.JavaVM)(unsafe.Pointer(app.JavaVM()))
}

func getFieldID(env *C.JNIEnv, class C.jclass, name, sig string) C.jfieldID {
	f := C.CString(name)
	defer C.free(unsafe.Pointer(f))
	s := C.CString(sig)
	defer C.free(unsafe.Pointer(s))
	jf := C.jni_GetFieldID(env, class, f, s)
	if err := exception(env); err != nil {
		panic(err)
	}
	return jf
}

func getMethodID(env *C.JNIEnv, class C.jclass, method, sig string) C.jmethodID {
	m := C.CString(method)
	defer C.free(unsafe.Pointer(m))
	s := C.CString(sig)
	defer C.free(unsafe.Pointer(s))
	jm := C.jni_GetMethodID(env, class, m, s)
	if err := exception(env); err != nil {
		panic(err)
	}
	return jm
}

func javaString(env *C.JNIEnv, str string) C.jstring {
	utf16Chars := utf16.Encode([]rune(str))
	var ptr *C.jchar
	if len(utf16Chars) > 0 {
		ptr = (*C.jchar)(unsafe.Pointer(&utf16Chars[0]))
	}
	return C.jni_NewString(env, ptr, C.int(len(utf16Chars)))
}

func varArgs(args []jvalue) *C.jvalue {
	if len(args) == 0 {
		return nil
	}
	return (*C.jvalue)(unsafe.Pointer(&args[0]))
}

func callVoidMethod(env *C.JNIEnv, obj C.jobject, method C.jmethodID, args ...jvalue) error {
	C.jni_CallVoidMethodA(env, obj, method, varArgs(args))
	return exception(env)
}

func callIntMethod(env *C.JNIEnv, obj C.jobject, method C.jmethodID, args ...jvalue) (int32, error) {
	res := C.jni_CallIntMethodA(env, obj, method, varArgs(args))
	return int32(res), exception(env)
}

func callFloatMethod(env *C.JNIEnv, obj C.jobject, method C.jmethodID, args ...jvalue) (float32, error) {
	res := C.jni_CallFloatMethodA(env, obj, method, varArgs(args))
	return float32(res), exception(env)
}

func callObjectMethod(env *C.JNIEnv, obj C.jobject, method C.jmethodID, args ...jvalue) (C.jobject, error) {
	res := C.jni_CallObjectMethodA(env, obj, method, varArgs(args))
	return res, exception(env)
}

func getObjectArrayElement(env *C.JNIEnv, jarr C.jobjectArray, index C.jsize) (C.jobject, error) {
	jobj := C.jni_GetObjectArrayElement(env, jarr, index)
	return jobj, exception(env)
}

func getObjectArrayLength(env *C.JNIEnv, jarr C.jobjectArray) int {
	return int(C.jni_GetArrayLength(env, C.jarray(jarr)))
}

// exception returns an error corresponding to the pending
// exception, or nil if no exception is pending. The pending
// exception is cleared.
func exception(env *C.JNIEnv) error {
	thr := C.jni_ExceptionOccurred(env)
	if thr == 0 {
		return nil
	}
	C.jni_ExceptionClear(env)
	cls := getObjectClass(env, C.jobject(thr))
	toString := getMethodID(env, cls, "toString", "()Ljava/lang/String;")
	msg, err := callObjectMethod(env, C.jobject(thr), toString)
	if err != nil {
		return err
	}
	return errors.New(goString(env, C.jstring(msg)))
}

func getObjectClass(env *C.JNIEnv, obj C.jobject) C.jclass {
	if obj == 0 {
		panic("null object")
	}
	cls := C.jni_GetObjectClass(env, C.jobject(obj))
	if err := exception(env); err != nil {
		// GetObjectClass should never fail.
		panic(err)
	}
	return cls
}

// goString converts the JVM jstring to a Go string.
func goString(env *C.JNIEnv, str C.jstring) string {
	if str == 0 {
		return ""
	}
	strlen := C.jni_GetStringLength(env, C.jstring(str))
	chars := C.jni_GetStringChars(env, C.jstring(str))
	utf16Chars := unsafe.Slice((*uint16)(unsafe.Pointer(chars)), strlen)
	utf8 := utf16.Decode(utf16Chars)
	return string(utf8)
}

func findClass(env *C.JNIEnv, name string) C.jclass {
	cn := C.CString(name)
	defer C.free(unsafe.Pointer(cn))
	return C.jni_FindClass(env, cn)
}

func setIntField(env *C.JNIEnv, obj C.jobject, field C.jfieldID, value int32) {
	C.jni_SetIntField(env, obj, field, C.jint(value))
}
