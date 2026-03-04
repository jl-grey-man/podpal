#!/usr/bin/env python3
"""
patch_rockbox_logo.py — Patch a custom boot logo into a Rockbox binary.

Replaces the compiled-in boot logo in rockbox.ipod with a custom image,
and recalculates the Rockbox checksum so the bootloader accepts it.

How it works:
  The Rockbox boot logo is compiled into the binary as raw pixel data
  (not a BMP file). This tool generates a reference logo from the original
  Rockbox source BMP, finds that exact byte pattern in the binary, replaces
  it with the user's image converted to the same native format, and fixes
  the checksum.

Rockbox binary format:
  - Bytes 0-3: checksum (big-endian uint32) = model_number + sum(payload_bytes)
  - Bytes 4-7: model ID string (e.g. "ipvd" for iPod Video)
  - Bytes 8+:  firmware payload

Native bitmap formats (per model):
  - Format 4: little-endian RGB565  (iPod Video, Nano 2G, Classic)
  - Format 5: byte-swapped RGB565   (iPod Color, Nano 1G)
  - Format 6: greyscale iPod 4-grey (iPod 1G/2G, 3G, 4G, Mini 1G/2G)

Usage:
    python3 patch_rockbox_logo.py <rockbox.ipod> <image_file> [--model ipodvideo]
    python3 patch_rockbox_logo.py /Volumes/IPOD/.rockbox/rockbox.ipod photo.jpg
    python3 patch_rockbox_logo.py rockbox.ipod logo.png --model ipodnano2g

Supports input images in any format PIL can read (JPEG, PNG, BMP, etc).
Automatically crops, resizes, and converts to the correct native format.
"""

import sys
import struct
import argparse
import os


# Per-model configuration derived from Rockbox source (tools/configure + config/*.h)
# model_id: the 4-byte string in the .ipod header (from scramble -add=XXXX)
# model_num: checksum seed (from ipodpatcher.c / tools/scramble)
# lcd_width/lcd_height: screen dimensions
# logo_width/logo_height: boot logo dimensions (from apps/bitmaps/native/SOURCES)
# bmp_format: bmp2rb format number (4=LE RGB565, 5=BE RGB565, 6=greyscale 4-grey)
IPOD_MODELS = {
    'ipodvideo': {
        'model_id': b'ipvd',
        'model_num': 5,
        'lcd_width': 320, 'lcd_height': 240,
        'logo_width': 320, 'logo_height': 98,
        'bmp_format': 4,
        'lcd_depth': 16,
        'description': 'iPod Video (5th/5.5th Gen)',
    },
    'ipod6g': {
        'model_id': b'ip6g',
        'model_num': 71,
        'lcd_width': 320, 'lcd_height': 240,
        'logo_width': 320, 'logo_height': 98,
        'bmp_format': 4,
        'lcd_depth': 16,
        'description': 'iPod Classic (6th/6.5th/7th Gen)',
    },
    'ipodnano2g': {
        'model_id': b'nn2g',
        'model_num': 62,
        'lcd_width': 176, 'lcd_height': 132,
        'logo_width': 176, 'logo_height': 54,
        'bmp_format': 4,
        'lcd_depth': 16,
        'description': 'iPod Nano 2nd Gen',
    },
    'ipodcolor': {
        'model_id': b'ipcl',  # scramble -add=ipco but model ID is ipcl
        'model_num': 8,
        'lcd_width': 220, 'lcd_height': 176,
        'logo_width': 220, 'logo_height': 68,
        'bmp_format': 5,
        'lcd_depth': 16,
        'description': 'iPod Color/Photo',
    },
    'ipodnano1g': {
        'model_id': b'nano',
        'model_num': 4,
        'lcd_width': 176, 'lcd_height': 132,
        'logo_width': 176, 'logo_height': 54,
        'bmp_format': 5,
        'lcd_depth': 16,
        'description': 'iPod Nano 1st Gen',
    },
    'ipod3g': {
        'model_id': b'ip3g',
        'model_num': 7,
        'lcd_width': 160, 'lcd_height': 128,
        'logo_width': 160, 'logo_height': 53,
        'bmp_format': 6,
        'lcd_depth': 2,
        'description': 'iPod 3rd Gen',
    },
    'ipod4g': {
        'model_id': b'ip4g',
        'model_num': 9,
        'lcd_width': 160, 'lcd_height': 128,
        'logo_width': 160, 'logo_height': 53,
        'bmp_format': 6,
        'lcd_depth': 2,
        'description': 'iPod 4th Gen (greyscale)',
    },
    'ipodmini1g': {
        'model_id': b'mini',
        'model_num': 3,
        'lcd_width': 138, 'lcd_height': 110,
        'logo_width': 138, 'logo_height': 46,
        'bmp_format': 6,
        'lcd_depth': 2,
        'description': 'iPod Mini 1st Gen',
    },
    'ipodmini2g': {
        'model_id': b'mn2g',
        'model_num': 11,
        'lcd_width': 138, 'lcd_height': 110,
        'logo_width': 138, 'logo_height': 46,
        'bmp_format': 6,
        'lcd_depth': 2,
        'description': 'iPod Mini 2nd Gen',
    },
    'ipod1g2g': {
        'model_id': b'1g2g',
        'model_num': 19,
        'lcd_width': 160, 'lcd_height': 128,
        'logo_width': 160, 'logo_height': 53,
        'bmp_format': 6,
        'lcd_depth': 2,
        'description': 'iPod 1st/2nd Gen',
    },
}


def detect_model(model_id_bytes):
    """Detect iPod model from the 4-byte model ID in the binary header."""
    for name, info in IPOD_MODELS.items():
        if info['model_id'] == bytes(model_id_bytes):
            return name, info
    return None, None


def image_to_native(img, logo_w, logo_h, bmp_format):
    """Convert a PIL Image to Rockbox native bitmap format.

    Crops to the logo aspect ratio, resizes, and converts to the correct
    pixel format for the target model.

    Args:
        img: PIL Image
        logo_w: target logo width
        logo_h: target logo height
        bmp_format: Rockbox bmp2rb format number
            4 = little-endian RGB565
            5 = big-endian (byte-swapped) RGB565
            6 = greyscale iPod 4-grey (2 pixels per byte, column-packed)

    Returns:
        bytes of native bitmap data
    """
    from PIL import Image

    img = img.convert('RGB')
    src_w, src_h = img.size

    # Crop to target aspect ratio
    target_ratio = logo_w / logo_h
    src_ratio = src_w / src_h
    if src_ratio > target_ratio:
        new_w = int(src_h * target_ratio)
        left = (src_w - new_w) // 2
        img = img.crop((left, 0, left + new_w, src_h))
    else:
        new_h = int(src_w / target_ratio)
        top = (src_h - new_h) // 2
        img = img.crop((0, top, src_w, top + new_h))

    img = img.resize((logo_w, logo_h), Image.LANCZOS)

    if bmp_format == 4:
        # Little-endian RGB565: each pixel is 2 bytes, LE
        data = bytearray()
        for y in range(logo_h):
            for x in range(logo_w):
                r, g, b = img.getpixel((x, y))
                rgb565 = ((r >> 3) << 11) | ((g >> 2) << 5) | (b >> 3)
                data += struct.pack('<H', rgb565)
        return bytes(data)

    elif bmp_format == 5:
        # Byte-swapped RGB565: each pixel is 2 bytes, BE
        data = bytearray()
        for y in range(logo_h):
            for x in range(logo_w):
                r, g, b = img.getpixel((x, y))
                rgb565 = ((r >> 3) << 11) | ((g >> 2) << 5) | (b >> 3)
                data += struct.pack('>H', rgb565)
        return bytes(data)

    elif bmp_format == 6:
        # Greyscale iPod 4-grey: 2bpp, column-packed
        # dst_w = (width + 3) / 4, dst_h = height, each entry is unsigned short
        # Pixel brightness mapped to 2-bit grey, packed 4 pixels per byte
        dst_w = (logo_w + 3) // 4
        dst = [0] * (dst_w * logo_h)
        for y in range(logo_h):
            for x in range(logo_w):
                r, g, b = img.getpixel((x, y))
                brightness = (3 * r + 6 * g + b) // 10
                grey_2bit = (~brightness & 0xC0) >> (2 * (x & 3))
                dst[y * dst_w + (x // 4)] |= grey_2bit
        data = bytearray()
        for val in dst:
            data += struct.pack('<H', val & 0xFFFF)
        return bytes(data)

    else:
        raise ValueError(f"Unsupported bmp_format: {bmp_format}")


def calc_native_logo_size(logo_w, logo_h, bmp_format):
    """Calculate the expected byte size of a native logo."""
    if bmp_format in (4, 5):
        return logo_w * logo_h * 2  # 2 bytes per pixel
    elif bmp_format == 6:
        dst_w = (logo_w + 3) // 4
        return dst_w * logo_h * 2  # unsigned short per entry
    else:
        raise ValueError(f"Unsupported bmp_format: {bmp_format}")


def find_logo_in_payload(payload, reference_logo):
    """Find the logo data in the payload by searching for a unique non-zero chunk.

    Uses a 64-byte chunk from the middle of the reference logo as a needle,
    then verifies the full match.

    Returns:
        offset into payload where logo starts, or None
    """
    # Find a non-trivial (non-zero) region to search for
    logo_len = len(reference_logo)
    mid = logo_len // 2

    # Try to find a 64-byte all-nonzero chunk near the middle
    needle = None
    needle_offset = None
    for start in range(mid, min(mid + 2000, logo_len - 64), 2):
        chunk = reference_logo[start:start + 64]
        if len(chunk) == 64 and all(b != 0 for b in chunk):
            needle = chunk
            needle_offset = start
            break

    if needle is None:
        # Fall back: use first 64 bytes that have at least some non-zero
        for start in range(0, logo_len - 64, 2):
            chunk = reference_logo[start:start + 64]
            if len(chunk) == 64 and any(b != 0 for b in chunk):
                needle = chunk
                needle_offset = start
                break

    if needle is None:
        return None

    pos = payload.find(needle)
    if pos == -1:
        return None

    logo_start = pos - needle_offset
    if logo_start < 0:
        return None

    # Verify full match
    candidate = payload[logo_start:logo_start + logo_len]
    if candidate == reference_logo:
        return logo_start

    # Partial match — still return if >95% matches (compression artifacts etc.)
    match_count = sum(1 for a, b in zip(candidate, reference_logo) if a == b)
    match_pct = match_count / logo_len * 100
    if match_pct > 95:
        return logo_start

    return None


def generate_reference_logo(model_info):
    """Generate the reference logo from Rockbox source BMP if available.

    Looks for the original rockboxlogo BMP in the Rockbox source tree.
    Returns the native format bytes, or None if source not available.
    """
    from PIL import Image

    logo_w = model_info['logo_width']
    logo_h = model_info['logo_height']
    depth = model_info['lcd_depth']
    fmt = model_info['bmp_format']

    # Try to find the source BMP
    bmp_name = f"rockboxlogo.{logo_w}x{logo_h}x{depth}.bmp"
    search_paths = [
        f"/tmp/rockbox-src/apps/bitmaps/native/{bmp_name}",
        os.path.join(os.path.dirname(__file__), '..', 'assets', bmp_name),
        os.path.join(os.path.dirname(__file__), bmp_name),
    ]

    for path in search_paths:
        if os.path.exists(path):
            img = Image.open(path)
            return image_to_native(img, logo_w, logo_h, fmt)

    return None


def calc_rockbox_checksum(payload, model_num):
    """Calculate Rockbox firmware checksum.

    checksum = model_number + sum(all payload bytes), as uint32.
    """
    checksum = model_num
    for b in payload:
        checksum = (checksum + b) & 0xFFFFFFFF
    return checksum


def patch_rockbox_logo(rockbox_path, image_path, model_name=None, backup=True):
    """Patch a Rockbox binary with a custom boot logo.

    Steps:
      1. Read the binary, parse header (checksum + model ID)
      2. Auto-detect or use specified model to get logo dimensions and format
      3. Generate reference logo from Rockbox source BMP
      4. Find the reference logo bytes in the payload
      5. Convert user's image to the same native format (same byte count)
      6. Replace the logo bytes in-place
      7. Recalculate and write the checksum

    Args:
        rockbox_path: Path to rockbox.ipod file
        image_path: Path to image file (any format PIL supports)
        model_name: Model name (e.g. 'ipodvideo'). Auto-detected if None.
        backup: Create .bak backup before patching

    Returns:
        dict with 'success' bool and details
    """
    from PIL import Image

    with open(rockbox_path, 'rb') as f:
        data = bytearray(f.read())

    original_size = len(data)

    # Parse header
    original_checksum = struct.unpack_from('>I', data, 0)[0]
    model_id = data[4:8]
    payload = data[8:]

    # Detect or look up model
    if model_name:
        if model_name not in IPOD_MODELS:
            return {'success': False, 'error': f"Unknown model '{model_name}'. "
                    f"Known: {', '.join(IPOD_MODELS.keys())}"}
        model_info = IPOD_MODELS[model_name]
        detected_name = model_name
    else:
        detected_name, model_info = detect_model(model_id)
        if model_info is None:
            model_id_str = model_id.decode('ascii', errors='replace')
            return {'success': False,
                    'error': f"Unknown model ID '{model_id_str}'. "
                    f"Use --model to specify. Known: {', '.join(IPOD_MODELS.keys())}"}

    print(f"[INFO] Model: {model_info['description']} ({detected_name})")
    print(f"[INFO] Model ID: {model_id.decode('ascii', errors='replace')}, "
          f"checksum seed: {model_info['model_num']}")
    print(f"[INFO] LCD: {model_info['lcd_width']}x{model_info['lcd_height']}, "
          f"logo: {model_info['logo_width']}x{model_info['logo_height']}, "
          f"format: {model_info['bmp_format']}")
    print(f"[INFO] Stored checksum: 0x{original_checksum:08X}")

    # Verify checksum
    verify_sum = calc_rockbox_checksum(payload, model_info['model_num'])
    if verify_sum != original_checksum:
        print(f"[WARN] Checksum mismatch: stored=0x{original_checksum:08X}, "
              f"calculated=0x{verify_sum:08X}")
        print(f"[WARN] Binary may already be corrupted!")
        return {'success': False,
                'error': 'Checksum mismatch — restore from a clean rockbox.ipod first'}
    print(f"[INFO] Checksum verified OK")

    # Generate reference logo
    ref_logo = generate_reference_logo(model_info)
    if ref_logo is None:
        return {'success': False,
                'error': f"Cannot find Rockbox source BMP for {detected_name}. "
                f"Need rockboxlogo.{model_info['logo_width']}x{model_info['logo_height']}"
                f"x{model_info['lcd_depth']}.bmp in /tmp/rockbox-src/"}

    expected_size = calc_native_logo_size(
        model_info['logo_width'], model_info['logo_height'], model_info['bmp_format'])
    print(f"[INFO] Reference logo: {len(ref_logo)} bytes (expected {expected_size})")

    # Find logo in payload
    logo_offset = find_logo_in_payload(bytes(payload), ref_logo)
    if logo_offset is None:
        return {'success': False, 'error': 'Could not find logo data in binary. '
                'The binary may use a different Rockbox version than the source BMPs.'}

    print(f"[INFO] Found logo at payload offset 0x{logo_offset:X} "
          f"(file offset 0x{logo_offset + 8:X})")

    # Verify the logo fits within the payload
    if logo_offset + len(ref_logo) > len(payload):
        return {'success': False,
                'error': f'Logo at 0x{logo_offset:X} extends beyond payload end'}

    # Convert user's image to native format
    img = Image.open(image_path)
    new_logo = image_to_native(
        img, model_info['logo_width'], model_info['logo_height'], model_info['bmp_format'])

    if len(new_logo) != len(ref_logo):
        return {'success': False,
                'error': f'New logo size {len(new_logo)} != reference {len(ref_logo)}'}

    # Backup
    if backup:
        bak_path = rockbox_path + '.bak'
        if not os.path.exists(bak_path):
            with open(bak_path, 'wb') as f:
                f.write(data)
            print(f"[INFO] Backup saved to {bak_path}")

    # Patch
    payload[logo_offset:logo_offset + len(new_logo)] = new_logo
    print(f"[INFO] Patched {len(new_logo)} bytes at payload offset 0x{logo_offset:X}")

    # Recalculate checksum
    new_checksum = calc_rockbox_checksum(payload, model_info['model_num'])
    struct.pack_into('>I', data, 0, new_checksum)
    data[8:] = payload

    assert len(data) == original_size, \
        f"File size changed: {original_size} -> {len(data)}"

    print(f"[INFO] Checksum: 0x{original_checksum:08X} -> 0x{new_checksum:08X}")

    # Write
    with open(rockbox_path, 'wb') as f:
        f.write(data)

    # Final verification
    with open(rockbox_path, 'rb') as f:
        verify_data = f.read()
    v_stored = struct.unpack_from('>I', verify_data, 0)[0]
    v_calc = calc_rockbox_checksum(verify_data[8:], model_info['model_num'])
    if v_stored != v_calc:
        print(f"[ERR] Post-write verification FAILED: 0x{v_stored:08X} != 0x{v_calc:08X}")
        return {'success': False, 'error': 'Post-write checksum verification failed'}

    print(f"[INFO] Post-write verification: OK")
    print(f"[OK] Successfully patched {rockbox_path}")
    return {
        'success': True,
        'model': detected_name,
        'logo_offset': logo_offset,
        'logo_size': len(new_logo),
        'old_checksum': original_checksum,
        'new_checksum': new_checksum,
    }


def main():
    parser = argparse.ArgumentParser(
        description='Patch a custom boot logo into a Rockbox binary.',
        epilog='Supported models: ' + ', '.join(IPOD_MODELS.keys())
    )
    parser.add_argument('rockbox', help='Path to rockbox.ipod file')
    parser.add_argument('image', help='Path to image file (JPEG, PNG, BMP, etc.)')
    parser.add_argument('--model', choices=list(IPOD_MODELS.keys()),
                        help='iPod model (auto-detected from binary if omitted)')
    parser.add_argument('--no-backup', action='store_true',
                        help='Skip creating .bak backup')
    parser.add_argument('--list-models', action='store_true',
                        help='List all supported models and exit')

    args = parser.parse_args()

    if args.list_models:
        print("Supported iPod models:")
        for name, info in IPOD_MODELS.items():
            print(f"  {name:15s}  {info['description']:35s}  "
                  f"{info['lcd_width']}x{info['lcd_height']}  "
                  f"logo {info['logo_width']}x{info['logo_height']}  "
                  f"{'color' if info['lcd_depth'] == 16 else 'greyscale'}")
        sys.exit(0)

    if not os.path.exists(args.rockbox):
        print(f"[ERR] File not found: {args.rockbox}")
        sys.exit(1)
    if not os.path.exists(args.image):
        print(f"[ERR] File not found: {args.image}")
        sys.exit(1)

    result = patch_rockbox_logo(
        args.rockbox, args.image,
        model_name=args.model,
        backup=not args.no_backup
    )

    if not result['success']:
        print(f"[ERR] {result['error']}")
        sys.exit(1)


if __name__ == '__main__':
    main()
