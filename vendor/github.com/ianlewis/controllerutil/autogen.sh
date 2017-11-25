#!/bin/sh
#
# Copyright 2017 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Runs autogen on all source files. This will add the proper licence headers to files that don't already have it.
# Requires https://github.com/mbrukman/autogen
find . -path ./vendor -prune -o -type f \( -iname \*.go -o -iname \*.yaml -o -iname \*.sh \) -print0 | xargs -0 grep -Le "Copyright [0-9][0-9][0-9][0-9] Google LLC" | xargs -r -L1 autogen -c "Google LLC" -l apache2 --no-tlc -i
