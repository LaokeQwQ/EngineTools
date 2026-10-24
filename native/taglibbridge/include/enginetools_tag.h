/*
 * EngineTools TagLib bridge -- stable C ABI.
 *
 * All functions return an integer status: 0 = success, non-zero = failure
 * category (see et_status). Diagnostic details are returned through the
 * out-error buffer as UTF-8 JSON {"code":"...","detail":"..."}.
 *
 * Every et_buffer returned by this library owns its memory; the caller
 * MUST release it exactly once with et_buffer_free. Buffers may be empty
 * ({NULL,0}) even on success.
 *
 * Paths are UTF-16 (Windows wide chars). No C++ type crosses the ABI.
 */
#ifndef ENGINETOOLS_TAG_H
#define ENGINETOOLS_TAG_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct et_buffer {
    uint8_t *data;
    size_t   len;
} et_buffer;

/* Status categories shared with the Go error mapping. */
enum et_status {
    ET_OK                 = 0,
    ET_ERR_OPEN           = 1, /* file missing/unreadable        */
    ET_ERR_FORMAT         = 2, /* unsupported/corrupt container  */
    ET_ERR_PATCH          = 3, /* patch JSON invalid             */
    ET_ERR_APPLY          = 4, /* tag write refused              */
    ET_ERR_IO             = 5, /* temp/replace IO problem        */
    ET_ERR_NOMEM          = 6,
    ET_ERR_INTERNAL       = 7
};

/* Capability + readable-metadata snapshot as JSON. */
int et_tag_probe(const wchar_t *path, et_buffer *json, et_buffer *error);

/* Full canonical Document as JSON. */
int et_tag_read(const wchar_t *path, et_buffer *json, et_buffer *error);

/* Applies a tri-state Patch JSON; returns WriteResult JSON. */
int et_tag_apply(const wchar_t *path,
                 const uint8_t *patch, size_t patch_len,
                 et_buffer *json, et_buffer *error);

/* SHA-256 over the AUDIO PAYLOAD ONLY: every tag chunk/frame/padding is
 * excluded by rebuilding the audio stream through TagLib's own container
 * APIs. Never decodes or reinterprets audio samples. */
int et_audio_payload_sha256(const wchar_t *path,
                            uint8_t out_hash[32],
                            et_buffer *error);

void et_buffer_free(et_buffer *value);

#ifdef __cplusplus
}
#endif
#endif /* ENGINETOOLS_TAG_H */
