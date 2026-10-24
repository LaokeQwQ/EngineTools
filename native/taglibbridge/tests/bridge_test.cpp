#include <windows.h>

#include <cstdio>
#include <cstring>
#include <iostream>
#include <string>

#include "enginetools_tag.h"

namespace {

struct Buffer {
    et_buffer value{};
    ~Buffer() { et_buffer_free(&value); }
};

bool contains(const et_buffer& buffer, const char* text) {
    if (!buffer.data || buffer.len == 0) return false;
    const std::string value(reinterpret_cast<const char*>(buffer.data), buffer.len);
    return value.find(text) != std::string::npos;
}

bool writeMinimalMPEG(const std::wstring& path) {
    FILE* file = nullptr;
    if (_wfopen_s(&file, path.c_str(), L"wb") != 0 || !file) return false;
    const unsigned char header[] = {0xFF, 0xFB, 0x90, 0x00};
    const unsigned char payload[413] = {};
    bool ok = true;
    for (int frame = 0; frame < 64 && ok; ++frame) {
        ok = std::fwrite(header, 1, sizeof(header), file) == sizeof(header) &&
             std::fwrite(payload, 1, sizeof(payload), file) == sizeof(payload);
    }
    ok = std::fclose(file) == 0 && ok;
    return ok;
}

bool check(bool condition, const char* message) {
    if (!condition) std::cerr << "bridge_test: " << message << '\n';
    return condition;
}

std::wstring temporaryMP3Path() {
    wchar_t directory[MAX_PATH] = {};
    if (GetTempPathW(MAX_PATH, directory) == 0) return {};
    wchar_t temporary[MAX_PATH] = {};
    if (GetTempFileNameW(directory, L"etb", 0, temporary) == 0) return {};
    DeleteFileW(temporary);
    return std::wstring(temporary) + L".mp3";
}

} // namespace

int main() {
    const std::wstring path = temporaryMP3Path();
    if (!check(!path.empty() && writeMinimalMPEG(path), "could not create MPEG fixture")) return 1;

    const auto cleanup = [&]() { DeleteFileW(path.c_str()); };
    bool ok = true;

    {
        Buffer json, error;
        ok &= check(et_tag_probe(path.c_str(), &json.value, &error.value) == ET_OK, "probe failed");
        ok &= check(contains(json.value, "\"format\":\"mp3\""), "probe did not report MP3");
    }
    {
        Buffer json, error;
        ok &= check(et_tag_read(path.c_str(), &json.value, &error.value) == ET_OK, "initial read failed");
		ok &= check(contains(json.value, "\"format\":\"mp3\""), "read did not report MP3");
    }

    unsigned char before[32] = {};
    {
        Buffer error;
        ok &= check(et_audio_payload_sha256(path.c_str(), before, &error.value) == ET_OK, "initial payload hash failed");
    }
    {
        static const char patch[] = "{\"title\":{\"mode\":\"set\",\"value\":\"Bridge Test\"}}";
        Buffer json, error;
        ok &= check(et_tag_apply(path.c_str(), reinterpret_cast<const uint8_t*>(patch), sizeof(patch) - 1,
                                 &json.value, &error.value) == ET_OK,
                    "apply failed");
        ok &= check(contains(json.value, "\"ok\":true"), "apply did not report success");
    }
    {
        Buffer json, error;
        ok &= check(et_tag_read(path.c_str(), &json.value, &error.value) == ET_OK, "post-apply read failed");
        ok &= check(contains(json.value, "Bridge Test"), "post-apply read lost title");
    }

    unsigned char after[32] = {};
    {
        Buffer error;
        ok &= check(et_audio_payload_sha256(path.c_str(), after, &error.value) == ET_OK, "post-apply payload hash failed");
        ok &= check(std::memcmp(before, after, sizeof(before)) == 0, "metadata write changed audio payload");
    }
    {
        static const char truncated[] = "{\"title\":{\"mode\":\"set\",\"value\":\"bad\"}";
        Buffer json, error;
        ok &= check(et_tag_apply(path.c_str(), reinterpret_cast<const uint8_t*>(truncated), sizeof(truncated) - 1,
                                 &json.value, &error.value) == ET_ERR_PATCH,
                    "truncated JSON was accepted");
    }
    {
        static const char trailing[] = "{} trailing";
        Buffer json, error;
        ok &= check(et_tag_apply(path.c_str(), reinterpret_cast<const uint8_t*>(trailing), sizeof(trailing) - 1,
                                 &json.value, &error.value) == ET_ERR_PATCH,
                    "trailing JSON bytes were accepted");
    }
    {
        static const char negTotal[] = "{\"track\":{\"mode\":\"set\",\"value\":{\"value\":1,\"total\":-1}}}";
        Buffer json, error;
        ok &= check(et_tag_apply(path.c_str(), reinterpret_cast<const uint8_t*>(negTotal), sizeof(negTotal) - 1,
                                 &json.value, &error.value) == ET_ERR_PATCH,
                    "negative total track patch was accepted");
    }
    {
        static const char totalTooSmall[] = "{\"track\":{\"mode\":\"set\",\"value\":{\"value\":5,\"total\":3}}}";
        Buffer json, error;
        ok &= check(et_tag_apply(path.c_str(), reinterpret_cast<const uint8_t*>(totalTooSmall), sizeof(totalTooSmall) - 1,
                                 &json.value, &error.value) == ET_ERR_PATCH,
                    "track value exceeding total was accepted");
    }

    cleanup();
    return ok ? 0 : 1;
}
