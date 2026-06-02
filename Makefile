CXX := g++
CXXFLAGS := -std=c++17 -Wall -O2

RAYLIB_DIR := /tmp/raylib/src

INCLUDES := -I$(RAYLIB_DIR)
LDFLAGS := $(RAYLIB_DIR)/libraylib.a -lGL -lX11 -lXrandr -lXinerama -lXi -lXcursor -lpthread -ldl -lrt -lm

TARGET := sumi
SRC := main.cpp

$(TARGET): $(SRC)
	$(CXX) $(CXXFLAGS) $(INCLUDES) $(SRC) -o $(TARGET) $(LDFLAGS)

clean:
	rm -f $(TARGET)

.PHONY: clean
