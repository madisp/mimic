LOCAL_PATH := $(call my-dir)

include $(CLEAR_VARS)

LOCAL_MODULE := mkfifo
LOCAL_SRC_FILES := mkfifo.cpp
LOCAL_CPPFLAGS := -std=gnu++0x -Wall
LOCAL_LDLIBS := -L$(SYSROOT)/usr/lib -llog

include $(BUILD_EXECUTABLE)
