# Third-party notices (repository root)

## TagLib (native metadata engine)

The audio metadata editor links the upstream TagLib library at runtime
through enginetools_tag.dll (see native/taglibbridge).

- Upstream: https://github.com/TagLib/taglib
- Pinned release: 2.0.2 -- archive SHA-256 recorded in
  build/taglib/taglib-pin.json and verified before extraction by
  build/windows/build-taglib.ps1.
- Consumed unmodified as a dynamic library; no TagLib source is committed.

TagLib is distributed under either of the following licenses, at the
recipient's choice:

### GNU Lesser General Public License v2.1

This library is free software; you can redistribute it and/or modify it
under the terms of the GNU Lesser General Public License version 2.1 as
published by the Free Software Foundation, with the additional permission
that linking the library statically or dynamically with other modules is
making a combined work based on the library. Full text:
https://www.gnu.org/licenses/old-licenses/lgpl-2.1.html

### Mozilla Public License v1.1

Alternatively, the contents may be used under the terms of the Mozilla
Public License Version 1.1 (https://www.mozilla.org/MPL/1.1/).
