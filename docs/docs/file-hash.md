---
sidebar_position: 7
---

# File Hash and Verification

## Overview

Postie implements a robust and fast file hashing system to ensure file integrity throughout the posting and downloading process. This document explains how the file hash is calculated, stored, and verified.

## Hash Generation Process

When generating NZB files, Postie includes a hash for each file. The hash is calculated using the following process:

1. Each article (segment) of the file is individually hashed using SHA256
2. All article hashes are combined and hashed again to create a final file hash

This approach provides several benefits:

- Complete file integrity verification (all parts of the file are hashed)
- Efficient computation during the posting process

The SHA256 algorithm is used for hashing, which provides a good balance between speed and collision resistance. Since the entire file content is included in the hash calculation (through all its articles), the hash is highly reliable for file verification and deduplication purposes.

## Verifying File Hash

The file hash is included in the generated NZB file and can be used by NZB clients to verify file integrity after downloading. To manually verify a file hash:

1. Split the file into the same size segments as specified in the NZB file
2. Calculate SHA256 hash for each segment
3. Concatenate all segment hashes
4. Calculate SHA256 hash of the combined string
5. Compare with the hash in the NZB file

This verification process ensures that all parts of the file have been downloaded correctly and haven't been tampered with.
