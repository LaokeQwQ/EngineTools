// EngineTools TagLib bridge.
//
// Implements the stable C ABI from include/enginetools_tag.h on top of
// upstream TagLib. Design notes:
//
//  * JSON output is built by hand (compact, UTF-8) -- no third-party JSON
//    dependency crosses the DLL boundary.
//  * A minimal recursive-descent parser handles the flat tri-state Patch
//    documents produced by the Go client; unknown fields are rejected so a
//    newer client cannot silently no-op against an older helper.
//  * Audio payload hashing strips EVERY tag through TagLib's own container
//    APIs into a sibling temp file, hashes that, then deletes it. Audio
//    samples are never decoded or reinterpreted.
//  * A compact public-domain-style SHA-256 ships inline so the ABI can keep
//    its fixed out_hash[32] contract without pulling OpenSSL.

#include <taglib/fileref.h>
#include <taglib/tag.h>
#include <taglib/tpropertymap.h>
#include <taglib/tvariant.h>
#include <taglib/tpicturetype.h>
#include <taglib/mpegfile.h>
#include <taglib/id3v1tag.h>
#include <taglib/id3v2frame.h>
#include <taglib/id3v2framefactory.h>
#include <taglib/id3v2tag.h>
#include <taglib/attachedpictureframe.h>
#include <taglib/popularimeterframe.h>
#include <taglib/textidentificationframe.h>
#include <taglib/unsynchronizedlyricsframe.h>
#include <taglib/commentsframe.h>
#include <taglib/flacfile.h>
#include <taglib/xiphcomment.h>
#include <taglib/wavfile.h>
#include <taglib/aifffile.h>

#include <windows.h>

#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <cerrno>
#include <climits>
#include <cstdint>
#include <map>
#include <memory>
#include <set>
#include <utility>
#include <string>
#include <vector>

#include "enginetools_tag.h"

namespace {

constexpr char Q = '"';

// ---------------------------- tiny SHA-256 ------------------------------

class Sha256 {
public:
    Sha256() { reset(); }
    void reset() {
        h_[0]=0x6a09e667; h_[1]=0xbb67ae85; h_[2]=0x3c6ef372; h_[3]=0xa54ff53a;
        h_[4]=0x510e527f; h_[5]=0x9b05688c; h_[6]=0x1f83d9ab; h_[7]=0x5be0cd19;
        len_=0; buf_len_=0;
    }
    void update(const unsigned char* data, size_t n) {
        len_ += n;
        while (n > 0) {
            size_t take = (64 - buf_len_ < n) ? 64 - buf_len_ : n;
            memcpy(buf_ + buf_len_, data, take);
            buf_len_ += take; data += take; n -= take;
            if (buf_len_ == 64) { block(buf_); buf_len_ = 0; }
        }
    }
    void final(unsigned char out[32]) {
        unsigned char pad = 0x80;
        unsigned long long bits = len_ * 8ull;
        update(&pad, 1);
        unsigned char z = 0;
        while (buf_len_ != 56) update(&z, 1);
        unsigned char tail[8];
        for (int i = 0; i < 8; ++i) tail[i] = (unsigned char)(bits >> (56 - 8*i));
        update(tail, 8);
        for (int i = 0; i < 8; ++i) {
            out[i*4+0]=(unsigned char)(h_[i]>>24); out[i*4+1]=(unsigned char)(h_[i]>>16);
            out[i*4+2]=(unsigned char)(h_[i]>>8);  out[i*4+3]=(unsigned char)(h_[i]);
        }
    }
private:
    static unsigned int rotr(unsigned int x, int r){return (x>>r)|(x<<(32-r));}
    void block(const unsigned char* p) {
        static const unsigned int K[64]={
        0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5,0x3956c25b,0x59f111f1,
        0x923f82a4,0xab1c5ed5,0xd807aa98,0x12835b01,0x243185be,0x550c7dc3,
        0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174,0xe49b69c1,0xefbe4786,
        0x0fc19dc6,0x240ca1cc,0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da,
        0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7,0xc6e00bf3,0xd5a79147,
        0x06ca6351,0x14292967,0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13,
        0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85,0xa2bfe8a1,0xa81a664b,
        0xc24b8b70,0xc76c51a3,0xd192e819,0xd6990624,0xf40e3585,0x106aa070,
        0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5,0x391c0cb3,0x4ed8aa4a,
        0x5b9cca4f,0x682e6ff3,0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208,
        0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2};
        unsigned int w[64];
        for (int i=0;i<16;++i)
            w[i]=(p[i*4]<<24)|(p[i*4+1]<<16)|(p[i*4+2]<<8)|p[i*4+3];
        for (int i=16;i<64;++i){
            unsigned int s0=rotr(w[i-15],7)^rotr(w[i-15],18)^(w[i-15]>>3);
            unsigned int s1=rotr(w[i-2],17)^rotr(w[i-2],19)^(w[i-2]>>10);
            w[i]=w[i-16]+s0+w[i-7]+s1;
        }
        unsigned int a=h_[0],b=h_[1],c=h_[2],d=h_[3],e=h_[4],f=h_[5],g=h_[6],hh=h_[7];
        for (int i=0;i<64;++i){
            unsigned int S1=rotr(e,6)^rotr(e,11)^rotr(e,25);
            unsigned int ch=(e&f)^((~e)&g);
            unsigned int t1=hh+S1+ch+K[i]+w[i];
            unsigned int S0=rotr(a,2)^rotr(a,13)^rotr(a,22);
            unsigned int mj=(a&b)^(a&c)^(b&c);
            unsigned int t2=S0+mj;
            hh=g; g=f; f=e; e=d+t1; d=c; c=b; b=a; a=t1+t2;
        }
        h_[0]+=a; h_[1]+=b; h_[2]+=c; h_[3]+=d;
        h_[4]+=e; h_[5]+=f; h_[6]+=g; h_[7]+=hh;
    }
    unsigned int h_[8];
    unsigned long long len_;
    unsigned char buf_[64];
    size_t buf_len_;
};

bool hashFileBytes(const std::wstring& path, unsigned char out[32]) {
    FILE* f = nullptr;
    if (_wfopen_s(&f, path.c_str(), L"rb") != 0 || !f) return false;
    Sha256 sha;
    char buf[65536];
    size_t n;
    while ((n = fread(buf, 1, sizeof(buf), f)) > 0) sha.update((unsigned char*)buf, n);
    fclose(f);
    sha.final(out);
    return true;
}

// ------------------------- buffer / error plumbing ----------------------

void set_buffer(et_buffer* out, const void* data, size_t len) {
    if (!out) return;
    out->data = nullptr; out->len = 0;
    if (len == 0 || !data) return;
    out->data = (uint8_t*)malloc(len);
    if (!out->data) { out->len = (size_t)-1; return; }
    memcpy(out->data, data, len);
    out->len = len;
}

std::string jsonEscape(const std::string& in) {
    std::string out;
    out.reserve(in.size() + 16);
    for (unsigned char c : in) {
        switch (c) {
            case '"':  out += "\\\""; break;
            case '\\': out += "\\\\"; break;
            case '\b': out += "\\b"; break;
            case '\f': out += "\\f"; break;
            case '\n': out += "\\n"; break;
            case '\r': out += "\\r"; break;
            case '\t': out += "\\t"; break;
            default:
                if (c < 0x20) {
                    char t[8];
                    snprintf(t, sizeof(t), "\\u%04x", c);
                    out += t;
                } else out += (char)c;
        }
    }
    return out;
}

// --------------------------- JSON output --------------------------------

void putStr(std::string& j, const char* key, const std::string& v,
            bool* first, bool omitEmpty) {
    if (omitEmpty && v.empty()) return;
    if (!*first) j += ",";
    *first = false;
    j += '"'; j += key; j += "\":";
    j += '"'; j += jsonEscape(v); j += '"';
}

void putInt(std::string& j, const char* key, long long v, bool* first) {
    if (!*first) j += ",";
    *first = false;
    j += '"'; j += key; j += "\":";
    char t[24]; snprintf(t, sizeof(t), "%lld", v); j += t;
}

void putNumPair(std::string& j, const char* key, int value, int total, bool* first) {
    if (!*first) j += ",";
    *first = false;
    j += '"' ; j += key; j += "\":{";
    j += "\"value\":" ;
    char t[16]; snprintf(t, sizeof(t), "%d", value); j += t;
    if (total > 0) { j += ",\"total\":"; snprintf(t, sizeof(t), "%d", total); j += t; }
    j += "}";
}

void putArr(std::string& j, const char* key, const std::vector<std::string>& v, bool* first) {
    if (!*first) j += ",";
    *first = false;
    j += '"'; j += key; j += "\":[";
    for (size_t i = 0; i < v.size(); ++i) {
        if (i) j += ",";
        j += '"'; j += jsonEscape(v[i]); j += '"';
    }
    j += "]";
}

bool isJSONNumber(const std::string& value) {
    // PropertyMap values are arbitrary strings.  Validate the complete JSON
    // number grammar before inserting one into hand-built output; merely
    // checking the character set would allow invalid forms such as "+1",
    // "1.", or "-.5" to escape as malformed JSON.
    if (value.empty()) return false;
    size_t i = 0;
    if (value[i] == '-') ++i;
    if (i >= value.size()) return false;
    if (value[i] == '0') {
        ++i;
        if (i < value.size() && value[i] >= '0' && value[i] <= '9') return false;
    } else if (value[i] >= '1' && value[i] <= '9') {
        while (i < value.size() && value[i] >= '0' && value[i] <= '9') ++i;
    } else {
        return false;
    }
    if (i < value.size() && value[i] == '.') {
        ++i;
        const size_t fractionStart = i;
        while (i < value.size() && value[i] >= '0' && value[i] <= '9') ++i;
        if (i == fractionStart) return false;
    }
    if (i < value.size() && (value[i] == 'e' || value[i] == 'E')) {
        ++i;
        if (i < value.size() && (value[i] == '+' || value[i] == '-')) ++i;
        const size_t exponentStart = i;
        while (i < value.size() && value[i] >= '0' && value[i] <= '9') ++i;
        if (i == exponentStart) return false;
    }
    return i == value.size();
}

std::string num(const std::string& v) {
    return isJSONNumber(v) ? v : "0";
}

std::pair<std::string, std::string> splitNumberPair(const std::string& value,
                                                     const std::string& total) {
    const size_t slash = value.find('/');
    if (slash == std::string::npos)
        return {value, total};
    const std::string first = value.substr(0, slash);
    const std::string second = total.empty() ? value.substr(slash + 1) : total;
    return {first, second};
}

// --------------------------- error plumbing -----------------------------

int fail(et_buffer* err, int code, const char* codeName, const std::string& detail) {
    if (err) {
        std::string j = "{\"code\":\"";
        j += codeName; j += "\",\"detail\":\"";
        j += jsonEscape(detail);
        j += "\"}";
        set_buffer(err, j.data(), j.size());
    }
    return code;
}

// --------------------------- TagLib helpers -----------------------------

std::string fmtToFormat(TagLib::File* f) {
    if (dynamic_cast<TagLib::MPEG::File*>(f)) return "mp3";
    if (dynamic_cast<TagLib::FLAC::File*>(f)) return "flac";
    if (dynamic_cast<TagLib::RIFF::WAV::File*>(f)) return "wav";
    if (dynamic_cast<TagLib::RIFF::AIFF::File*>(f)) return "aiff";
    return "unknown";
}

std::vector<std::string> propsList(const TagLib::PropertyMap& pm, const char* key) {
    std::vector<std::string> out;
    auto it = pm.find(key);
    if (it != pm.end())
        for (const auto& v : it->second) out.push_back(v.to8Bit(true));
    return out;
}

std::string propsOne(const TagLib::PropertyMap& pm, const char* key) {
    auto v = propsList(pm, key);
    return v.empty() ? std::string() : v.front();
}

std::string base64String(const TagLib::ByteVector& bytes) {
    const TagLib::ByteVector encoded = bytes.toBase64();
    return encoded.isEmpty() ? std::string() : std::string(encoded.data(), encoded.size());
}

int readRating(TagLib::File* f) {
    auto* mpeg = dynamic_cast<TagLib::MPEG::File*>(f);
    if (!mpeg || !mpeg->hasID3v2Tag()) return 0;
    const auto frames = mpeg->ID3v2Tag()->frameList("POPM");
    for (const auto* frame : frames) {
        const auto* popm = dynamic_cast<const TagLib::ID3v2::PopularimeterFrame*>(frame);
        if (!popm) continue;
        int raw = popm->rating();
        if (raw < 0) raw = 0;
        if (raw > 255) raw = 255;
        return (raw * 100 + 127) / 255;
    }
    return 0;
}

std::string pictureTypeName(int type) {
    if (type < 0 || type > 0x14) return std::string();
    return TagLib::Utils::pictureTypeToString(type).to8Bit(true);
}

// --------------------------- minimal JSON -------------------------------

struct JVal {
    enum Kind { NUL, BOOL, NUM, STR, ARR, OBJ } kind = NUL;
    bool b = false;
    std::string s;                       // NUM keeps raw text
    std::vector<JVal> arr;
    std::map<std::string, JVal> obj;
    bool has(const std::string& k) const { return obj.count(k) != 0; }
};

bool parseInt(const JVal& value, int* out) {
    if (!out || value.kind != JVal::NUM || value.s.empty()) return false;
    char* end = nullptr;
    errno = 0;
    long parsed = std::strtol(value.s.c_str(), &end, 10);
    if (errno == ERANGE || end == value.s.c_str() || *end != '\0' || parsed < INT_MIN || parsed > INT_MAX)
        return false;
    *out = static_cast<int>(parsed);
    return true;
}

class JsonParser {
public:
    JsonParser(const char* data, size_t length)
        : p_(data), end_(data ? data + length : data) {
        if (!data) valid_ = false;
    }

    JVal parse() {
        skip();
        if (!valid_ || p_ == end_) {
            fail();
            return JVal{};
        }
        JVal result = value();
        skip();
        // A JSON document must contain exactly one value. In particular,
        // reject a valid prefix followed by arbitrary bytes.
        if (p_ != end_) fail();
        return result;
    }

    bool valid() const { return valid_; }

private:
    const char* p_ = nullptr;
    const char* end_ = nullptr;
    bool valid_ = true;
    size_t depth_ = 0;

    static constexpr size_t kMaxDepth = 64;

    class DepthGuard {
    public:
        explicit DepthGuard(JsonParser* parser) : parser_(parser), active_(parser_->enter()) {}
        ~DepthGuard() { if (active_) parser_->leave(); }
        bool active() const { return active_; }
    private:
        JsonParser* parser_;
        bool active_;
    };

    void fail() { valid_ = false; }

    bool enter() {
        if (++depth_ > kMaxDepth) {
            --depth_;
            fail();
            return false;
        }
        return true;
    }

    void leave() { --depth_; }

    static bool isDigit(char c) { return c >= '0' && c <= '9'; }
    static int hexValue(char c) {
        if (c >= '0' && c <= '9') return c - '0';
        if (c >= 'a' && c <= 'f') return c - 'a' + 10;
        if (c >= 'A' && c <= 'F') return c - 'A' + 10;
        return -1;
    }

    void skip() {
        while (valid_ && p_ && p_ < end_ && (*p_ == 0x20 || *p_ == 0x09 || *p_ == 0x0A || *p_ == 0x0D)) ++p_;
    }

    bool eat(char c) {
        skip();
        if (valid_ && p_ && p_ < end_ && *p_ == c) {
            ++p_;
            return true;
        }
        return false;
    }

    bool literal(const char* text, size_t length) {
        if (!valid_ || !p_ || !end_ || static_cast<size_t>(end_ - p_) < length || std::memcmp(p_, text, length) != 0)
            return false;
        p_ += length;
        return true;
    }

    static void appendCodepoint(std::string& out, uint32_t codepoint) {
        if (codepoint <= 0x7F) {
            out.push_back(static_cast<char>(codepoint));
        } else if (codepoint <= 0x7FF) {
            out.push_back(static_cast<char>(0xC0 | (codepoint >> 6)));
            out.push_back(static_cast<char>(0x80 | (codepoint & 0x3F)));
        } else if (codepoint <= 0xFFFF) {
            out.push_back(static_cast<char>(0xE0 | (codepoint >> 12)));
            out.push_back(static_cast<char>(0x80 | ((codepoint >> 6) & 0x3F)));
            out.push_back(static_cast<char>(0x80 | (codepoint & 0x3F)));
        } else {
            out.push_back(static_cast<char>(0xF0 | (codepoint >> 18)));
            out.push_back(static_cast<char>(0x80 | ((codepoint >> 12) & 0x3F)));
            out.push_back(static_cast<char>(0x80 | ((codepoint >> 6) & 0x3F)));
            out.push_back(static_cast<char>(0x80 | (codepoint & 0x3F)));
        }
    }

    bool unicodeEscape(uint32_t* codepoint) {
        if (!codepoint || p_ >= end_ || *p_ != 'u' || static_cast<size_t>(end_ - p_) < 5) {
            fail();
            return false;
        }
        ++p_; // 'u'
        uint32_t value = 0;
        for (int i = 0; i < 4; ++i) {
            const int digit = hexValue(*p_++);
            if (digit < 0) {
                fail();
                return false;
            }
            value = (value << 4) | static_cast<uint32_t>(digit);
        }
        if (value >= 0xD800 && value <= 0xDBFF) {
            // JSON represents non-BMP characters as a UTF-16 surrogate pair.
            if (static_cast<size_t>(end_ - p_) < 6 || p_[0] != '\\' || p_[1] != 'u') {
                fail();
                return false;
            }
            p_ += 2;
            uint32_t low = 0;
            for (int i = 0; i < 4; ++i) {
                const int digit = hexValue(*p_++);
                if (digit < 0) {
                    fail();
                    return false;
                }
                low = (low << 4) | static_cast<uint32_t>(digit);
            }
            if (low < 0xDC00 || low > 0xDFFF) {
                fail();
                return false;
            }
            value = 0x10000 + ((value - 0xD800) << 10) + (low - 0xDC00);
        } else if (value >= 0xDC00 && value <= 0xDFFF) {
            fail();
            return false;
        }
        *codepoint = value;
        return true;
    }

    bool stringLit(std::string* out) {
        if (!out || !eat(0x22)) {
            fail();
            return false;
        }
        out->clear();
        while (p_ < end_) {
            const unsigned char c = static_cast<unsigned char>(*p_++);
            if (c == 0x22) return true;
            if (c < 0x20) {
                fail();
                return false;
            }
            if (c != 0x5C) {
                out->push_back(static_cast<char>(c));
                continue;
            }
            if (p_ >= end_) {
                fail();
                return false;
            }
            const char escaped = *p_++;
            switch (escaped) {
                case 0x22: out->push_back(0x22); break;
                case 0x5C: out->push_back(0x5C); break;
                case '/':  out->push_back('/'); break;
                case 'b':  out->push_back(static_cast<char>(8)); break;
                case 'f':  out->push_back(static_cast<char>(12)); break;
                case 'n':  out->push_back(static_cast<char>(10)); break;
                case 'r':  out->push_back(static_cast<char>(13)); break;
                case 't':  out->push_back(static_cast<char>(9)); break;
                case 'u': {
                    // Move back so unicodeEscape can consume the leading 'u'.
                    --p_;
                    uint32_t codepoint = 0;
                    if (!unicodeEscape(&codepoint)) return false;
                    appendCodepoint(*out, codepoint);
                    break;
                }
                default:
                    fail();
                    return false;
            }
        }
        fail(); // unterminated string
        return false;
    }

    JVal number() {
        JVal v;
        v.kind = JVal::NUM;
        const char* start = p_;
        if (p_ < end_ && *p_ == '-') ++p_;
        if (p_ >= end_) {
            fail();
            return JVal{};
        }
        if (*p_ == '0') {
            ++p_;
            if (p_ < end_ && isDigit(*p_)) {
                fail();
                return JVal{};
            }
        } else if (*p_ >= '1' && *p_ <= '9') {
            while (p_ < end_ && isDigit(*p_)) ++p_;
        } else {
            fail();
            return JVal{};
        }
        if (p_ < end_ && *p_ == '.') {
            ++p_;
            if (p_ >= end_ || !isDigit(*p_)) {
                fail();
                return JVal{};
            }
            while (p_ < end_ && isDigit(*p_)) ++p_;
        }
        if (p_ < end_ && (*p_ == 'e' || *p_ == 'E')) {
            ++p_;
            if (p_ < end_ && (*p_ == '+' || *p_ == '-')) ++p_;
            if (p_ >= end_ || !isDigit(*p_)) {
                fail();
                return JVal{};
            }
            while (p_ < end_ && isDigit(*p_)) ++p_;
        }
        v.s.assign(start, static_cast<size_t>(p_ - start));
        return v;
    }

    JVal value() {
        skip();
        if (!valid_ || p_ >= end_) {
            fail();
            return JVal{};
        }
        switch (*p_) {
            case 0x7B: return object();
            case 0x5B: return array();
            case 0x22: {
                JVal v;
                v.kind = JVal::STR;
                if (!stringLit(&v.s)) return JVal{};
                return v;
            }
            default: break;
        }
        if (*p_ == 't' && literal("true", 4)) {
            JVal v; v.kind = JVal::BOOL; v.b = true; return v;
        }
        if (*p_ == 'f' && literal("false", 5)) {
            JVal v; v.kind = JVal::BOOL; v.b = false; return v;
        }
        if (*p_ == 'n' && literal("null", 4)) return JVal{};
        if (*p_ == '-' || isDigit(*p_)) return number();
        fail();
        return JVal{};
    }

    JVal object() {
        DepthGuard depth(this);
        if (!depth.active()) return JVal{};
        JVal v;
        v.kind = JVal::OBJ;
        ++p_; // '{'
        skip();
        if (eat(0x7D)) return v;
        for (;;) {
            skip();
            if (p_ >= end_ || *p_ != 0x22) {
                fail();
                return JVal{};
            }
            std::string key;
            if (!stringLit(&key)) return JVal{};
            if (v.obj.find(key) != v.obj.end()) {
                fail(); // duplicate keys are ambiguous for a patch document
                return JVal{};
            }
            if (!eat(0x3A)) {
                fail();
                return JVal{};
            }
            JVal child = value();
            if (!valid_) return JVal{};
            v.obj[std::move(key)] = std::move(child);
            skip();
            if (eat(0x7D)) return v;
            if (!eat(0x2C)) {
                fail();
                return JVal{};
            }
            skip();
            if (p_ >= end_ || *p_ == 0x7D) {
                fail(); // trailing comma
                return JVal{};
            }
        }
    }

    JVal array() {
        DepthGuard depth(this);
        if (!depth.active()) return JVal{};
        JVal v;
        v.kind = JVal::ARR;
        ++p_; // '['
        skip();
        if (eat(0x5D)) return v;
        for (;;) {
            if (p_ >= end_) {
                fail();
                return JVal{};
            }
            JVal child = value();
            if (!valid_) return JVal{};
            v.arr.push_back(std::move(child));
            skip();
            if (eat(0x5D)) return v;
            if (!eat(0x2C)) {
                fail();
                return JVal{};
            }
            skip();
            if (p_ >= end_ || *p_ == 0x5D) {
                fail(); // trailing comma
                return JVal{};
            }
        }
    }
};

// ------------------------- document building ----------------------------

std::string buildDocumentJSON(TagLib::File* f) {
    std::string j = "{\"format\":\"";
    j += fmtToFormat(f);
    j += "\",\"warnings\":[],";
    bool first = true;
    TagLib::PropertyMap pm = f->properties();
    if (auto* wav = dynamic_cast<TagLib::RIFF::WAV::File*>(f)) {
        if (wav->InfoTag()) {
            const TagLib::PropertyMap infoPm = wav->InfoTag()->properties();
            for (const auto& entry : infoPm) {
                if (!entry.second.isEmpty() && (pm.find(entry.first) == pm.end() || pm[entry.first].isEmpty())) {
                    pm[entry.first] = entry.second;
                }
            }
        }
    }

    auto putStr = [&](const char* key, const std::string& v, bool omitEmpty) {
        if (omitEmpty && v.empty()) return;
        if (!first) j += ",";
        first = false;
        j += Q; j += key; j += Q; j += ":"; j += Q;
        j += jsonEscape(v); j += Q;
    };
    auto putArrS = [&](const char* key, const std::vector<std::string>& v) {
        if (!first) j += ",";
        first = false;
        j += Q; j += key; j += Q; j += ":[";
        for (size_t i = 0; i < v.size(); ++i) {
            if (i) j += ",";
            j += Q; j += jsonEscape(v[i]); j += Q;
        }
        j += "]";
    };

    putStr("title", propsOne(pm, "TITLE"), true);
    putArrS("artists", propsList(pm, "ARTIST"));
    putStr("album", propsOne(pm, "ALBUM"), true);
    putArrS("albumArtists", propsList(pm, "ALBUMARTIST"));

    auto pairJson = [&](const char* key, const std::string& value, const std::string& total) {
        const auto parts = splitNumberPair(value, total);
        if (!first) j += ",";
        first = false;
        j += Q; j += key; j += Q; j += ":{\"value\":";
        j += parts.first.empty() ? "0" : num(parts.first);
        if (!parts.second.empty()) { j += ",\"total\":"; j += num(parts.second); }
        j += "}";
    };
    pairJson("track", propsOne(pm, "TRACKNUMBER"), propsOne(pm, "TRACKTOTAL"));
    pairJson("disc",  propsOne(pm, "DISCNUMBER"),  propsOne(pm, "DISCTOTAL"));

    putStr("date", propsOne(pm, "DATE"), true);
    putArrS("genres", propsList(pm, "GENRE"));
    putArrS("composers", propsList(pm, "COMPOSER"));
    putStr("comment", propsOne(pm, "COMMENT"), true);
    if (!first) j += ",";
    first = false;
    j += "\"bpm\":" + num(propsOne(pm, "BPM"));
    putStr("key", propsOne(pm, "INITIALKEY"), true);
    if (!first) j += ",";
    first = false;
    j += "\"rating\":" + std::to_string(readRating(f));
    putStr("isrc", propsOne(pm, "ISRC"), true);
    putStr("lyrics", propsOne(pm, "LYRICS"), true);

    // Pictures are exposed through TagLib's complex-property API so the same
    // code handles APIC and FLAC Picture blocks without dropping descriptors.
    // A malformed/native-unsupported Variant must not make the bridge emit an
    // invalid descriptor, so validate the required values before serializing.
    std::vector<std::string> pics;
    const auto complexPictures = f->complexProperties("PICTURE");
    const std::string idPrefix = fmtToFormat(f) == "flac" ? "flac-" : "apic-";
    int pictureIndex = 0;
    for (const auto& property : complexPictures) {
        bool dataOK = false;
        bool mimeOK = false;
        bool descriptionOK = false;
        bool typeOK = false;
        bool widthOK = false;
        bool heightOK = false;
        const TagLib::ByteVector data = property.value("data").value<TagLib::ByteVector>(&dataOK);
        const TagLib::String mime = property.value("mimeType").value<TagLib::String>(&mimeOK);
        const TagLib::String description = property.value("description").value<TagLib::String>(&descriptionOK);
        const TagLib::String pictureType = property.value("pictureType").value<TagLib::String>(&typeOK);
        const int width = property.value("width").value<int>(&widthOK);
        const int height = property.value("height").value<int>(&heightOK);
        const int type = typeOK ? TagLib::Utils::pictureTypeFromString(pictureType) : 0;
        if (!dataOK || !mimeOK || data.isEmpty() || mime.isEmpty()) continue;
        const std::string descriptionText = descriptionOK ? description.to8Bit(true) : std::string();
        std::string one = "{\"id\":\"" + idPrefix + std::to_string(pictureIndex++) +
                          "\",\"type\":" + std::to_string(type) +
                          ",\"mime\":\"" + jsonEscape(mime.to8Bit(true)) +
                          "\",\"description\":\"" + jsonEscape(descriptionText) +
                          "\",\"bytes\":" + std::to_string(data.size()) +
                          ",\"width\":" + std::to_string(widthOK && width >= 0 ? width : 0) +
                          ",\"height\":" + std::to_string(heightOK && height >= 0 ? height : 0) +
                          ",\"data\":\"" + jsonEscape(base64String(data)) + "\"}";
        pics.push_back(one);
    }
    if (!pics.empty()) {
        if (!first) j += ",";
        first = false;
        j += "\"pictures\":[";
        for (size_t i = 0; i < pics.size(); ++i) { if (i) j += ","; j += pics[i]; }
        j += "]";
    }

    j += "}";
    return j;
}

// ------------------------------ patch apply -----------------------------

struct ApplyOutcome {
    bool ok = false;
    int changed = 0;
    std::vector<std::string> warnings;
};

struct FieldSpec { const char* name; const char* prop; };

void markChanged(ApplyOutcome* out) {
    if (out) ++out->changed;
}

std::string modeOf(const JVal& field) {
    if (field.kind != JVal::OBJ) return "invalid";
    const auto it = field.obj.find("mode");
    if (it == field.obj.end()) return "unchanged";
    return it->second.kind == JVal::STR ? it->second.s : "invalid";
}

bool setProp(TagLib::PropertyMap& pm, const char* key, const std::string& v) {
    return pm.replace(key, TagLib::StringList(TagLib::String(v, TagLib::String::UTF8)));
}

void clearProp(TagLib::PropertyMap& pm, const char* key) { pm.erase(key); }

void removeDescriptionFrames(TagLib::ID3v2::Tag* tag, const char* frameID,
                             const char* canonicalDescription) {
    if (!tag) return;
    const TagLib::String canonical(canonicalDescription, TagLib::String::UTF8);
    std::vector<TagLib::ID3v2::Frame*> remove;
    for (auto* frame : tag->frameList(frameID)) {
        TagLib::String description;
        if (std::strcmp(frameID, "COMM") == 0) {
            if (auto* comment = dynamic_cast<TagLib::ID3v2::CommentsFrame*>(frame))
                description = comment->description();
        } else if (std::strcmp(frameID, "USLT") == 0) {
            if (auto* lyrics = dynamic_cast<TagLib::ID3v2::UnsynchronizedLyricsFrame*>(frame))
                description = lyrics->description();
        }
        if (description.isEmpty() || description.upper() == canonical)
            remove.push_back(frame);
    }
    for (auto* frame : remove) tag->removeFrame(frame);
}

bool removeID3PropertyFrames(TagLib::ID3v2::Tag* tag, const char* key) {
    if (!tag) return false;
    if (std::strcmp(key, "DATE") == 0) {
        tag->removeFrames("TDRC");
        tag->removeFrames("TRDA");
        tag->removeFrames("TDAT");
        tag->removeFrames("TYER");
        tag->removeFrames("TIME");
        return true;
    }
    if (std::strcmp(key, "COMMENT") == 0) {
        removeDescriptionFrames(tag, "COMM", "COMMENT");
        return true;
    }
    if (std::strcmp(key, "LYRICS") == 0) {
        removeDescriptionFrames(tag, "USLT", "LYRICS");
        return true;
    }
    const TagLib::ByteVector frameID = TagLib::ID3v2::Frame::keyToFrameID(
        TagLib::String(key, TagLib::String::UTF8));
    if (frameID.isEmpty()) return false;
    tag->removeFrames(frameID);
    return true;
}

bool setID3Property(TagLib::ID3v2::Tag* tag, const char* key,
                    const TagLib::StringList& values) {
    if (!tag || values.isEmpty() || !removeID3PropertyFrames(tag, key)) return false;
    auto* frame = TagLib::ID3v2::FrameFactory::instance()->createFrameForProperty(
        TagLib::String(key, TagLib::String::UTF8), values);
    if (!frame) return false;
    tag->addFrame(frame);
    return true;
}

bool parseTrackValue(const JVal& field, std::string* value) {
    if (!value || field.kind != JVal::OBJ) return false;
    const auto valueIt = field.obj.find("value");
    if (valueIt == field.obj.end() || valueIt->second.kind != JVal::NUM) return false;
    int number = 0;
    if (!parseInt(valueIt->second, &number) || number < 1) return false;
    *value = valueIt->second.s;
    const auto totalIt = field.obj.find("total");
    if (totalIt != field.obj.end()) {
        int total = 0;
        if (totalIt->second.kind != JVal::NUM || !parseInt(totalIt->second, &total) || total < 0) return false;
        if (total > 0) {
            if (number > total) return false;
            *value += "/" + totalIt->second.s;
        }
    }
    return true;
}

bool applyPropertyMapPatch(TagLib::PropertyMap& pm, const JVal& patch,
                           ApplyOutcome* out) {
    static const FieldSpec scalars[] = {
        {"title", "TITLE"}, {"album", "ALBUM"}, {"date", "DATE"},
        {"comment", "COMMENT"}, {"key", "INITIALKEY"}, {"isrc", "ISRC"},
        {"lyrics", "LYRICS"},
    };
    static const FieldSpec lists[] = {
        {"artists", "ARTIST"}, {"albumArtists", "ALBUMARTIST"},
        {"genres", "GENRE"}, {"composers", "COMPOSER"},
    };

    for (const auto& spec : scalars) {
        const auto it = patch.obj.find(spec.name);
        if (it == patch.obj.end()) continue;
        const std::string mode = modeOf(it->second);
        if (mode == "invalid") return false;
        if (mode == "unchanged" || mode.empty()) continue;
        if (mode == "clear") {
            clearProp(pm, spec.prop);
            markChanged(out);
            continue;
        }
        if (mode != "set" || it->second.kind != JVal::OBJ) return false;
        const auto valueIt = it->second.obj.find("value");
        if (valueIt == it->second.obj.end() || valueIt->second.kind != JVal::STR || valueIt->second.s.empty())
            return false;
        if (!setProp(pm, spec.prop, valueIt->second.s)) return false;
        markChanged(out);
    }

    for (const auto& spec : lists) {
        const auto it = patch.obj.find(spec.name);
        if (it == patch.obj.end()) continue;
        const std::string mode = modeOf(it->second);
        if (mode == "invalid") return false;
        if (mode == "unchanged" || mode.empty()) continue;
        if (mode == "clear") {
            clearProp(pm, spec.prop);
            markChanged(out);
            continue;
        }
        if (mode != "set" || it->second.kind != JVal::OBJ) return false;
        const auto valueIt = it->second.obj.find("value");
        if (valueIt == it->second.obj.end() || valueIt->second.kind != JVal::ARR || valueIt->second.arr.empty())
            return false;
        TagLib::StringList list;
        for (const auto& item : valueIt->second.arr) {
            if (item.kind != JVal::STR || item.s.empty()) return false;
            list.append(TagLib::String(item.s, TagLib::String::UTF8));
        }
        if (!pm.replace(spec.prop, list)) return false;
        markChanged(out);
    }

    for (const char* numericKey : {"track", "disc"}) {
        const auto it = patch.obj.find(numericKey);
        if (it == patch.obj.end()) continue;
        const std::string mode = modeOf(it->second);
        if (mode == "invalid") return false;
        const bool isTrack = std::strcmp(numericKey, "track") == 0;
        const char* target = isTrack ? "TRACKNUMBER" : "DISCNUMBER";
        const char* totalTarget = isTrack ? "TRACKTOTAL" : "DISCTOTAL";
        if (mode == "unchanged" || mode.empty()) continue;
        if (mode == "clear") {
            clearProp(pm, target);
            clearProp(pm, totalTarget);
            markChanged(out);
            continue;
        }
        if (mode != "set" || it->second.kind != JVal::OBJ) return false;
        const auto valueIt = it->second.obj.find("value");
        if (valueIt == it->second.obj.end() || valueIt->second.kind != JVal::NUM) return false;
        int value = 0;
        if (!parseInt(valueIt->second, &value) || value < 1) return false;
        if (!setProp(pm, target, valueIt->second.s)) return false;
        const auto totalIt = it->second.obj.find("total");
        clearProp(pm, totalTarget);
        if (totalIt != it->second.obj.end()) {
            int total = 0;
            if (totalIt->second.kind != JVal::NUM || !parseInt(totalIt->second, &total)) return false;
            if (total > 0 && !setProp(pm, totalTarget, totalIt->second.s)) return false;
        }
        markChanged(out);
    }

    const auto bpmIt = patch.obj.find("bpm");
    if (bpmIt != patch.obj.end()) {
        const std::string mode = modeOf(bpmIt->second);
        if (mode == "invalid") return false;
        if (mode == "set") {
            if (bpmIt->second.kind != JVal::OBJ) return false;
            const auto valueIt = bpmIt->second.obj.find("value");
            if (valueIt == bpmIt->second.obj.end() || valueIt->second.kind != JVal::NUM || valueIt->second.s.empty())
                return false;
            if (!setProp(pm, "BPM", valueIt->second.s)) return false;
            markChanged(out);
        } else if (mode == "clear") {
            clearProp(pm, "BPM");
            markChanged(out);
        } else if (mode != "unchanged" && !mode.empty()) {
            return false;
        }
    }
    return true;
}

bool applyID3PatchTo(TagLib::ID3v2::Tag* tag, const JVal& patch,
                     ApplyOutcome* out) {
    if (!tag) return false;
    static const FieldSpec scalars[] = {
        {"title", "TITLE"}, {"album", "ALBUM"}, {"date", "DATE"},
        {"comment", "COMMENT"}, {"key", "INITIALKEY"}, {"isrc", "ISRC"},
        {"lyrics", "LYRICS"},
    };
    static const FieldSpec lists[] = {
        {"artists", "ARTIST"}, {"albumArtists", "ALBUMARTIST"},
        {"genres", "GENRE"}, {"composers", "COMPOSER"},
    };

    for (const auto& spec : scalars) {
        const auto it = patch.obj.find(spec.name);
        if (it == patch.obj.end()) continue;
        const std::string mode = modeOf(it->second);
        if (mode == "invalid") return false;
        if (mode == "unchanged" || mode.empty()) continue;
        if (mode == "clear") {
            if (!removeID3PropertyFrames(tag, spec.prop)) return false;
            markChanged(out);
            continue;
        }
        if (mode != "set" || it->second.kind != JVal::OBJ) return false;
        const auto valueIt = it->second.obj.find("value");
        if (valueIt == it->second.obj.end() || valueIt->second.kind != JVal::STR || valueIt->second.s.empty())
            return false;
        TagLib::StringList values;
        values.append(TagLib::String(valueIt->second.s, TagLib::String::UTF8));
        if (!setID3Property(tag, spec.prop, values)) return false;
        markChanged(out);
    }

    for (const auto& spec : lists) {
        const auto it = patch.obj.find(spec.name);
        if (it == patch.obj.end()) continue;
        const std::string mode = modeOf(it->second);
        if (mode == "invalid") return false;
        if (mode == "unchanged" || mode.empty()) continue;
        if (mode == "clear") {
            if (!removeID3PropertyFrames(tag, spec.prop)) return false;
            markChanged(out);
            continue;
        }
        if (mode != "set" || it->second.kind != JVal::OBJ) return false;
        const auto valueIt = it->second.obj.find("value");
        if (valueIt == it->second.obj.end() || valueIt->second.kind != JVal::ARR || valueIt->second.arr.empty())
            return false;
        TagLib::StringList values;
        for (const auto& item : valueIt->second.arr) {
            if (item.kind != JVal::STR || item.s.empty()) return false;
            values.append(TagLib::String(item.s, TagLib::String::UTF8));
        }
        if (!setID3Property(tag, spec.prop, values)) return false;
        markChanged(out);
    }

    for (const char* numericKey : {"track", "disc"}) {
        const auto it = patch.obj.find(numericKey);
        if (it == patch.obj.end()) continue;
        const std::string mode = modeOf(it->second);
        if (mode == "invalid") return false;
        const char* target = std::strcmp(numericKey, "track") == 0 ? "TRACKNUMBER" : "DISCNUMBER";
        if (mode == "unchanged" || mode.empty()) continue;
        if (mode == "clear") {
            if (!removeID3PropertyFrames(tag, target)) return false;
            markChanged(out);
            continue;
        }
        if (mode != "set") return false;
        std::string value;
        if (!parseTrackValue(it->second, &value)) return false;
        TagLib::StringList values;
        values.append(TagLib::String(value, TagLib::String::UTF8));
        if (!setID3Property(tag, target, values)) return false;
        markChanged(out);
    }

    const auto bpmIt = patch.obj.find("bpm");
    if (bpmIt != patch.obj.end()) {
        const std::string mode = modeOf(bpmIt->second);
        if (mode == "invalid") return false;
        if (mode == "set") {
            if (bpmIt->second.kind != JVal::OBJ) return false;
            const auto valueIt = bpmIt->second.obj.find("value");
            if (valueIt == bpmIt->second.obj.end() || valueIt->second.kind != JVal::NUM || valueIt->second.s.empty())
                return false;
            TagLib::StringList values;
            values.append(TagLib::String(valueIt->second.s, TagLib::String::UTF8));
            if (!setID3Property(tag, "BPM", values)) return false;
            markChanged(out);
        } else if (mode == "clear") {
            if (!removeID3PropertyFrames(tag, "BPM")) return false;
            markChanged(out);
        } else if (mode != "unchanged" && !mode.empty()) {
            return false;
        }
    }
    return true;
}

unsigned int numberPrefix(const std::string& value) {
    char* end = nullptr;
    errno = 0;
    const unsigned long parsed = std::strtoul(value.c_str(), &end, 10);
    if (errno == ERANGE || end == value.c_str() || parsed > UINT_MAX) return 0;
    return static_cast<unsigned int>(parsed);
}

std::string firstStringValue(const JVal& field) {
    if (field.kind != JVal::OBJ) return {};
    const auto it = field.obj.find("value");
    if (it == field.obj.end() || it->second.kind != JVal::ARR || it->second.arr.empty()) return {};
    return it->second.arr.front().kind == JVal::STR ? it->second.arr.front().s : std::string();
}

void applyID3v1Patch(TagLib::ID3v1::Tag* tag, const JVal& patch) {
    if (!tag) return;
    auto stringField = [&](const char* name, void (TagLib::Tag::*setter)(const TagLib::String&)) {
        const auto it = patch.obj.find(name);
        if (it == patch.obj.end()) return;
        const std::string mode = modeOf(it->second);
        if (mode == "clear") {
            (tag->*setter)(TagLib::String());
        } else if (mode == "set" && it->second.kind == JVal::OBJ) {
            const auto valueIt = it->second.obj.find("value");
            if (valueIt != it->second.obj.end() && valueIt->second.kind == JVal::STR)
                (tag->*setter)(TagLib::String(valueIt->second.s, TagLib::String::UTF8));
        }
    };
    stringField("title", &TagLib::Tag::setTitle);
    stringField("album", &TagLib::Tag::setAlbum);
    stringField("comment", &TagLib::Tag::setComment);

    const auto artistIt = patch.obj.find("artists");
    if (artistIt != patch.obj.end()) {
        const std::string mode = modeOf(artistIt->second);
        if (mode == "clear") tag->setArtist(TagLib::String());
        else if (mode == "set") {
            const std::string value = firstStringValue(artistIt->second);
            if (!value.empty()) tag->setArtist(TagLib::String(value, TagLib::String::UTF8));
        }
    }
    const auto genreIt = patch.obj.find("genres");
    if (genreIt != patch.obj.end()) {
        const std::string mode = modeOf(genreIt->second);
        if (mode == "clear") tag->setGenre(TagLib::String());
        else if (mode == "set") {
            const std::string value = firstStringValue(genreIt->second);
            if (!value.empty()) tag->setGenre(TagLib::String(value, TagLib::String::UTF8));
        }
    }

    const auto dateIt = patch.obj.find("date");
    if (dateIt != patch.obj.end()) {
        const std::string mode = modeOf(dateIt->second);
        if (mode == "clear") tag->setYear(0);
        else if (mode == "set" && dateIt->second.kind == JVal::OBJ) {
            const auto valueIt = dateIt->second.obj.find("value");
            if (valueIt != dateIt->second.obj.end() && valueIt->second.kind == JVal::STR)
                tag->setYear(numberPrefix(valueIt->second.s));
        }
    }
    for (const char* name : {"track"}) {
        const auto it = patch.obj.find(name);
        if (it == patch.obj.end()) continue;
        const std::string mode = modeOf(it->second);
        if (mode == "clear") tag->setTrack(0);
        else if (mode == "set") {
            std::string value;
            if (parseTrackValue(it->second, &value)) tag->setTrack(numberPrefix(value));
        }
    }
}

bool applyRatingPatchTo(TagLib::File* f, const JVal& field, ApplyOutcome* out) {
	if (!out || field.kind != JVal::OBJ) return false;
    const std::string mode = modeOf(field);
	if (mode == "invalid") return false;
    if (mode == "unchanged" || mode.empty()) return true;
    auto* mpeg = dynamic_cast<TagLib::MPEG::File*>(f);
    if (!mpeg) return false;
    if (mode == "clear") {
        if (mpeg->hasID3v2Tag()) mpeg->ID3v2Tag()->removeFrames("POPM");
        ++out->changed;
        return true;
    }
    if (mode != "set") return false;
    auto valueIt = field.obj.find("value");
    int rating = 0;
    if (valueIt == field.obj.end() || !parseInt(valueIt->second, &rating) || rating < 0 || rating > 100)
        return false;
    auto* tag = mpeg->ID3v2Tag(true);
    TagLib::ID3v2::PopularimeterFrame* selected = nullptr;
    for (auto* frame : tag->frameList("POPM")) {
        selected = dynamic_cast<TagLib::ID3v2::PopularimeterFrame*>(frame);
        if (selected) break;
    }
    if (!selected) {
        selected = new TagLib::ID3v2::PopularimeterFrame();
        selected->setEmail("enginetools@local");
        tag->addFrame(selected);
    }
    selected->setRating((rating * 255 + 50) / 100);
    ++out->changed;
    return true;
}

constexpr size_t kMaxNativePictureBytes = 20u * 1024u * 1024u;

bool validBase64(const std::string& encoded) {
    if (encoded.empty() || encoded.size() % 4 != 0) return false;
    size_t padding = 0;
    if (encoded.back() == '=') {
        padding = 1;
        if (encoded.size() > 1 && encoded[encoded.size() - 2] == '=') padding = 2;
    }
    for (size_t i = 0; i < encoded.size() - padding; ++i) {
        const unsigned char c = static_cast<unsigned char>(encoded[i]);
        if (!(c >= 'A' && c <= 'Z') && !(c >= 'a' && c <= 'z') &&
            !(c >= '0' && c <= '9') && c != '+' && c != '/') return false;
    }
    for (size_t i = encoded.size() - padding; i < encoded.size(); ++i) {
        if (encoded[i] != '=') return false;
    }
    return true;
}

bool pictureBytesMatchMIME(const std::string& mime, const TagLib::ByteVector& data) {
    const size_t size = static_cast<size_t>(data.size());
    const auto byteAt = [&](size_t index) { return static_cast<unsigned char>(data[static_cast<unsigned int>(index)]); };
    if (mime == "image/jpeg") {
        return size >= 3 && byteAt(0) == 0xFF && byteAt(1) == 0xD8 && byteAt(2) == 0xFF;
    }
    if (mime == "image/png") {
        static const unsigned char signature[] = {0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A};
        if (size < sizeof(signature)) return false;
        for (size_t i = 0; i < sizeof(signature); ++i) {
            if (byteAt(i) != signature[i]) return false;
        }
        return true;
    }
    return false;
}

bool applyPicturesPatchTo(TagLib::File* f, const JVal& field, ApplyOutcome* out) {
    if (!f || !out || field.kind != JVal::OBJ) return false;
    const std::string mode = modeOf(field);
    if (mode == "invalid") return false;
    if (mode == "unchanged" || mode.empty()) return true;
    auto setPictures = [&](const TagLib::List<TagLib::VariantMap>& pictures) {
        // MPEG::File delegates complex properties through a TagUnion. That
        // does not create an ID3v2 tag for ID3v1-only files, so target the
        // owning tag directly and let the normal save path persist it.
        if (auto* mpeg = dynamic_cast<TagLib::MPEG::File*>(f)) {
            auto* tag = mpeg->ID3v2Tag(true);
            return tag && tag->setComplexProperties("PICTURE", pictures);
        }
        return f->setComplexProperties("PICTURE", pictures);
    };
    if (mode == "clear") {
		if (!setPictures({})) return false;
        ++out->changed;
        return true;
    }
    if (mode != "set") return false;
    auto valueIt = field.obj.find("value");
    if (valueIt == field.obj.end() || valueIt->second.kind != JVal::ARR || valueIt->second.arr.empty())
        return false;

    TagLib::List<TagLib::VariantMap> pictures;
    std::set<std::string> pictureIDs;
    for (const auto& item : valueIt->second.arr) {
        if (item.kind != JVal::OBJ) return false;
        const auto idIt = item.obj.find("id");
        const auto mimeIt = item.obj.find("mime");
        const auto dataIt = item.obj.find("data");
        const auto typeIt = item.obj.find("type");
        if (idIt == item.obj.end() || idIt->second.kind != JVal::STR || idIt->second.s.empty() ||
            !pictureIDs.insert(idIt->second.s).second ||
            mimeIt == item.obj.end() || mimeIt->second.kind != JVal::STR ||
            (mimeIt->second.s != "image/jpeg" && mimeIt->second.s != "image/png") ||
            dataIt == item.obj.end() || dataIt->second.kind != JVal::STR || dataIt->second.s.empty() ||
            typeIt == item.obj.end()) return false;
        int type = 0;
        if (!parseInt(typeIt->second, &type) || type < 0 || type > 0x14) return false;
        const size_t maxEncodedBytes = ((kMaxNativePictureBytes + 2) / 3) * 4;
        if (dataIt->second.s.size() > maxEncodedBytes || !validBase64(dataIt->second.s)) return false;
        const TagLib::ByteVector encoded(dataIt->second.s.data(), static_cast<unsigned int>(dataIt->second.s.size()));
        const TagLib::ByteVector data = TagLib::ByteVector::fromBase64(encoded);
        if (data.isEmpty() || static_cast<size_t>(data.size()) > kMaxNativePictureBytes ||
            !pictureBytesMatchMIME(mimeIt->second.s, data)) return false;

        TagLib::VariantMap picture;
        picture.insert("data", data);
        picture.insert("mimeType", TagLib::String(mimeIt->second.s, TagLib::String::UTF8));
        picture.insert("pictureType", TagLib::Utils::pictureTypeToString(type));
        const auto descriptionIt = item.obj.find("description");
        if (descriptionIt != item.obj.end() && descriptionIt->second.kind == JVal::STR)
            picture.insert("description", TagLib::String(descriptionIt->second.s, TagLib::String::UTF8));
        int dimension = 0;
        const auto widthIt = item.obj.find("width");
        if (widthIt != item.obj.end()) {
            if (!parseInt(widthIt->second, &dimension) || dimension < 0) return false;
            picture.insert("width", dimension);
        }
        dimension = 0;
        const auto heightIt = item.obj.find("height");
        if (heightIt != item.obj.end()) {
            if (!parseInt(heightIt->second, &dimension) || dimension < 0) return false;
            picture.insert("height", dimension);
        }
        pictures.append(picture);
    }
    if (!setPictures(pictures)) return false;
    ++out->changed;
    return true;
}

TagLib::ID3v2::Version saveVersionFor(const TagLib::ID3v2::Tag* tag,
                                      bool hadTag) {
    if (!hadTag || !tag || !tag->header()) return TagLib::ID3v2::v3;
    switch (tag->header()->majorVersion()) {
        // TagLib 2.x can read ID3v2.2 but only renders v2.3/v2.4.  Downgrade
        // an existing v2.2 tag to the closest writable representation rather
        // than referring to the non-existent ID3v2::v2 enum.
        case 2: return TagLib::ID3v2::v3;
        case 3: return TagLib::ID3v2::v3;
        default: return TagLib::ID3v2::v4;
    }
}

bool applyPatchTo(TagLib::File* f, const JVal& patch, ApplyOutcome* out) {
    if (!f || patch.kind != JVal::OBJ || !out) return false;

    static const char* const allowedFields[] = {
        "title", "artists", "album", "albumArtists", "track", "disc",
        "date", "genres", "composers", "comment", "bpm", "key",
        "rating", "isrc", "lyrics", "pictures",
    };
    for (const auto& entry : patch.obj) {
        bool known = false;
        for (const char* field : allowedFields) {
            if (entry.first == field) {
                known = true;
                break;
            }
        }
        if (!known) return false;
    }

    auto* mpeg = dynamic_cast<TagLib::MPEG::File*>(f);
    auto* wav = dynamic_cast<TagLib::RIFF::WAV::File*>(f);
    auto* aiff = dynamic_cast<TagLib::RIFF::AIFF::File*>(f);

    bool hadID3v2 = false;
    TagLib::ID3v2::Tag* id3 = nullptr;
    if (mpeg) {
        hadID3v2 = mpeg->hasID3v2Tag();
        id3 = mpeg->ID3v2Tag(true);
        if (!applyID3PatchTo(id3, patch, out)) return false;
        if (auto* v1 = mpeg->ID3v1Tag()) applyID3v1Patch(v1, patch);
    } else if (wav) {
        hadID3v2 = wav->hasID3v2Tag();
        id3 = wav->ID3v2Tag();
        if (!applyID3PatchTo(id3, patch, out)) return false;
        TagLib::PropertyMap info = wav->InfoTag()->properties();
        if (!applyPropertyMapPatch(info, patch, nullptr)) return false;
        const TagLib::PropertyMap rejected = wav->InfoTag()->setProperties(info);
        if (!rejected.isEmpty() || !rejected.unsupportedData().isEmpty()) {
            out->warnings.push_back("unsupported_properties");
            return false;
        }
    } else if (aiff) {
        hadID3v2 = aiff->hasID3v2Tag();
        id3 = aiff->tag();
        if (!applyID3PatchTo(id3, patch, out)) return false;
    } else {
        TagLib::PropertyMap pm = f->properties();
        if (!applyPropertyMapPatch(pm, patch, out)) return false;
        const TagLib::PropertyMap rejected = f->setProperties(pm);
        if (!rejected.isEmpty() || !rejected.unsupportedData().isEmpty()) {
            out->warnings.push_back("unsupported_properties");
            return false;
        }
    }

    const auto ratingIt = patch.obj.find("rating");
    if (ratingIt != patch.obj.end() && !applyRatingPatchTo(f, ratingIt->second, out)) return false;
    const auto picturesIt = patch.obj.find("pictures");
    if (picturesIt != patch.obj.end() && !applyPicturesPatchTo(f, picturesIt->second, out)) return false;

    if (mpeg) {
        out->ok = mpeg->save(TagLib::MPEG::File::AllTags, TagLib::File::StripOthers,
                             saveVersionFor(id3, hadID3v2), TagLib::File::DoNotDuplicate);
    } else if (wav) {
        out->ok = wav->save(TagLib::RIFF::WAV::File::AllTags,
                            TagLib::File::StripOthers,
                            saveVersionFor(id3, hadID3v2));
    } else if (aiff) {
        out->ok = aiff->save(saveVersionFor(id3, hadID3v2));
    } else {
        out->ok = f->save();
    }
    return out->ok;
}

// --------------------------- payload hashing ----------------------------

bool stripTagsToTemp(const wchar_t* src, std::wstring& tmpOut) {
    tmpOut = std::wstring(src) + L".audiotmp";
    if (!CopyFileW(src, tmpOut.c_str(), FALSE)) {
        tmpOut.clear();
        return false;
    }
    auto failAndRemove = [&]() {
        DeleteFileW(tmpOut.c_str());
        tmpOut.clear();
        return false;
    };
    TagLib::FileRef ref(tmpOut.c_str(), false);
    if (ref.isNull()) return failAndRemove();
    TagLib::File* f = ref.file();
    if (auto* m = dynamic_cast<TagLib::MPEG::File*>(f)) {
        m->strip(TagLib::MPEG::File::ID3v1 | TagLib::MPEG::File::ID3v2 |
                 TagLib::MPEG::File::APE);
        return m->save() ? true : failAndRemove();
    }
    if (auto* fl = dynamic_cast<TagLib::FLAC::File*>(f)) {
        fl->strip();
        fl->removePictures();
        return fl->save() ? true : failAndRemove();
    }
    if (auto* w = dynamic_cast<TagLib::RIFF::WAV::File*>(f)) {
        w->strip();
        return w->save() ? true : failAndRemove();
    }
    if (auto* a = dynamic_cast<TagLib::RIFF::AIFF::File*>(f)) {
        if (auto* tag = a->tag()) {
            auto frames = tag->frameList();
            for (auto* frame : frames)
                tag->removeFrame(frame);
        }
        return a->save() ? true : failAndRemove();
    }
    return failAndRemove();
}

// --------------------------- buffer/error -------------------------------

static void fill_buffer(et_buffer* out, const std::string& data) {
    if (!out) return;
    const size_t kMaxJson = 32u * 1024u * 1024u;
    if (data.size() > kMaxJson) { out->data = nullptr; out->len = (size_t)-1; return; }
    set_buffer(out, data.data(), data.size());
}

} // namespace

// ------------------------------ ABI exports -----------------------------

int et_tag_probe(const wchar_t* path, et_buffer* json, et_buffer* error) {
    try {
        TagLib::FileRef ref(path, false);
        if (ref.isNull())
            return fail(error, ET_ERR_OPEN, "open_failed", "TagLib could not parse this file");
        TagLib::File* f = ref.file();
        std::string j = "{\"format\":\"";
        j += fmtToFormat(f);
        j += "\",\"tagVersion\":\"present\"}";
        if (json) fill_buffer(json, j);
        return ET_OK;
    } catch (const std::exception& e) {
        return fail(error, ET_ERR_INTERNAL, "internal", e.what());
    }
}

int et_tag_read(const wchar_t* path, et_buffer* json, et_buffer* error) {
    try {
        TagLib::FileRef ref(path, false);
        if (ref.isNull())
            return fail(error, ET_ERR_OPEN, "open_failed", "TagLib could not parse this file");
        std::string doc = buildDocumentJSON(ref.file());
        if (json) fill_buffer(json, doc);
        return ET_OK;
    } catch (const std::exception& e) {
        return fail(error, ET_ERR_INTERNAL, "internal", e.what());
    }
}

int et_tag_apply(const wchar_t* path,
                 const uint8_t* patch, size_t patch_len,
                 et_buffer* json, et_buffer* error) {
    if (!patch || patch_len == 0)
        return fail(error, ET_ERR_PATCH, "patch_invalid", "empty patch buffer");
    constexpr size_t kMaxPatchJSON = 32u * 1024u * 1024u;
    if (patch_len > kMaxPatchJSON)
        return fail(error, ET_ERR_PATCH, "patch_invalid", "patch buffer exceeds the safety cap");
    try {
        JsonParser parser(reinterpret_cast<const char*>(patch), patch_len);
        JVal patchVal = parser.parse();
        if (!parser.valid() || patchVal.kind != JVal::OBJ)
            return fail(error, ET_ERR_PATCH, "patch_invalid", "patch is not an object");

        TagLib::FileRef ref(path, false);
        if (ref.isNull())
            return fail(error, ET_ERR_OPEN, "open_failed", "TagLib could not parse this file");
        TagLib::File* f = ref.file();

        ApplyOutcome outcome;
        bool dispatched = applyPatchTo(f, patchVal, &outcome);
        std::string j = "{\"ok\":" + std::string(dispatched ? "true" : "false") +
                        ",\"changed\":" + std::to_string(outcome.changed) + "}";
        if (json) fill_buffer(json, j);
        if (!dispatched || !outcome.ok)
            return fail(error, ET_ERR_APPLY, "apply_failed", "TagLib rejected metadata or could not save the file");
        return ET_OK;
    } catch (const std::exception& e) {
        return fail(error, ET_ERR_PATCH, "patch_invalid", e.what());
    }
}

int et_audio_payload_sha256(const wchar_t* path,
                            uint8_t out_hash[32],
                            et_buffer* error) {
    std::wstring tmp;
    if (!stripTagsToTemp(path, tmp))
        return fail(error, ET_ERR_APPLY, "payload_strip_failed",
                    "could not produce a tag-free copy of the audio stream");
    Sha256 sha;
    FILE* f = nullptr;
    if (_wfopen_s(&f, tmp.c_str(), L"rb") != 0 || !f) {
        DeleteFileW(tmp.c_str());
        return fail(error, ET_ERR_IO, "io_failed", "cannot reopen stripped audio");
    }
    char buf[65536];
    size_t n;
    while ((n = fread(buf, 1, sizeof(buf), f)) > 0) sha.update((unsigned char*)buf, n);
    fclose(f);
    DeleteFileW(tmp.c_str());
    memset(out_hash, 0, 32);
    sha.final(out_hash);
    return ET_OK;
}

void et_buffer_free(et_buffer *value) {
    if (!value) return;
    free(value->data);
    value->data = nullptr;
    value->len = 0;
}
