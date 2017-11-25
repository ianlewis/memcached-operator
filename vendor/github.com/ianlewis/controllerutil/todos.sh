#!/bin/bash -eu
#
# Copyright 2017 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Prints lines with TODO or FIXME in them.
find . -name vendor -prune -o -name .git -prune -o -path \*.git\* -prune -o -type f -exec grep -ne '\(FIXME\|TODO\):' {} /dev/null \; | sed -e 's/:[ 	]*\/\/[ 	]*/: /'
