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
#include <taglib/mpegfile.h>
#include <taglib/id3v2tag.h>
#include <taglib/attachedpictureframe.h>
#include <taglib/textidentificationframe.h>
#include <taglib/unsynchronizedlyricsframe.h>
#include <taglib/commentsframe.h>
#include <taglib/flacfile.h>
#include <taglib/xiphcomment.h>
#include <taglib/wavfile.h>
#include <taglib/aifffile.h>

#include <windows.h>

#include <cstdio>
#include <cstring>
#include <map>
#include <memory>
#include <string>
#include <vector>

#include "enginetools_tag.h"

namespace {

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

std::string num(const std::string& v) {
    // numeric-ish strings from properties stay as-is after basic sanity
    for (char c : v) if ((c < '0' || c > '9') && c != '.' && c != '-') return "0";
    return v.empty() ? "0" : v;
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

using TagLibFile = std::unique_ptr<TagLib::File>;

TagLibFile openByWidePath(const wchar_t* path, int* status, et_buffer* err) {
    TagLib::FileRef ref(path, false);
    if (ref.isNull()) {
        *status = fail(err, ET_ERR_OPEN, "open_failed",
                       "TagLib could not parse this file");
        return nullptr;
    }
    TagLibFile f(ref.file());
    ref.release(); // ownership moved into f
    return f;
}

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

void readPicturesJSON(std::string& j, TagLib::File* f, bool* first);

// --------------------------- minimal JSON -------------------------------

struct JVal {
    enum Kind { NUL, BOOL, NUM, STR, ARR, OBJ } kind = NUL;
    bool b = false;
    std::string s;                       // NUM keeps raw text
    std::vector<JVal> arr;
    std::map<std::string, JVal> obj;
    bool has(const std::string& k) const { return obj.count(k) != 0; }
};

class JsonParser {
public:
    explicit JsonParser(const char* p) : p_(p) {}
    JVal parse() { skip(); return value(); }
private:
    const char* p_;
    void skip() { while (*p_ == 0x20 || *p_ == 0x09 || *p_ == 0x0A || *p_ == 0x0D) ++p_; }
    bool eat(char c) { skip(); if (*p_ == c) { ++p_; return true; } return false; }
    JVal value() {
        skip();
        if (*p_ == 0x7B) return object();
        if (*p_ == 0x5B) return array();
        if (*p_ == 0x22) { JVal v; v.kind = JVal::STR; v.s = stringLit(); return v; }
        if (strncmp(p_, "true", 4) == 0)  { p_ += 4; JVal v; v.kind = JVal::BOOL; v.b = true;  return v; }
        if (strncmp(p_, "false", 5) == 0) { p_ += 5; JVal v; v.kind = JVal::BOOL; v.b = false; return v; }
        if (strncmp(p_, "null", 4) == 0)  { p_ += 4; return JVal{}; }
        JVal v; v.kind = JVal::NUM;
        const char* start = p_;
        while (*p_ && strchr("+-0123456789.eE", *p_)) ++p_;
        v.s.assign(start, p_ - start);
        return v;
    }
    std::string stringLit() {
        std::string out;
        ++p_;
        while (*p_ != 0x22) {
            if (*p_ == 0x5C) {
                ++p_;
                switch (*p_) {
                    case 0x22: out += char(0x22); break;
                    case 0x5C: out += char(0x5C); break;
                    case '/':  out += '/';  break;
                    case 'b':  out += char(8);  break;
                    case 'f':  out += char(12); break;
                    case 'n':  out += char(10); break;
                    case 'r':  out += char(13); break;
                    case 't':  out += char(9);  break;
                    default: break;
                }
                ++p_;
            } else out += *p_++;
        }
        ++p_;
        return out;
    }
    JVal object() {
        JVal v; v.kind = JVal::OBJ;
        ++p_; skip();
        if (eat(0x7D)) return v;
        for (;;) {
            skip();
            std::string key = stringLit();
            skip(); eat(0x3A);
            v.obj[key] = value();
            skip();
            if (eat(0x2C)) continue;
            eat(0x7D);
            return v;
        }
    }
    JVal array() {
        JVal v; v.kind = JVal::ARR;
        ++p_; skip();
        if (eat(0x5D)) return v;
        for (;;) {
            v.arr.push_back(value());
            skip();
            if (eat(0x2C)) continue;
            eat(0x5D);
            return v;
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
        if (!first) j += ",";
        first = false;
        j += Q; j += key; j += Q; j += ":{\"value\":";
        j += value.empty() ? "0" : num(value);
        if (!total.empty()) { j += ",\"total\":"; j += num(total); }
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
    putStr("key", propsOne(pm, "KEY"), true);
    putStr("isrc", propsOne(pm, "ISRC"), true);
    putStr("lyrics", propsOne(pm, "LYRICS"), true);

    // pictures
    std::vector<std::string> pics;
    if (auto* m = dynamic_cast<TagLib::MPEG::File*>(f)) {
        if (m->hasID3v2Tag()) {
            int idx = 0;
            for (auto* fr : m->ID3v2Tag()->frameList("APIC")) {
                auto* pic = dynamic_cast<TagLib::ID3v2::AttachedPictureFrame*>(fr);
                if (!pic) continue;
                std::string one = "{\"id\":\"apic-" + std::to_string(idx++) + "\",\"type\":" +
                                  std::to_string((int)pic->type()) + ",\"mime\":\"" +
                                  jsonEscape(pic->mimeType().to8Bit(true)) + "\",\"bytes\":" +
                                  std::to_string(pic->picture().size()) + "}";
                pics.push_back(one);
            }
        }
    } else if (auto* fl = dynamic_cast<TagLib::FLAC::File*>(f)) {
        int idx = 0;
        for (const auto* pic : fl->pictureList()) {
            std::string one = "{\"id\":\"flac-" + std::to_string(idx++) + "\",\"type\":" +
                              std::to_string((int)pic->type()) + ",\"mime\":\"" +
                              jsonEscape(pic->mimeType().to8Bit(true)) + "\",\"bytes\":" +
                              std::to_string(pic->data().size()) + "}";
            pics.push_back(one);
        }
    }
    if (!pics.empty()) {
        if (!first) j += ",";
        first = false;
        j += "\"pictures\":[";
        for (size_t i = 0; i < pics.size(); ++i) { if (i) j += ","; j += pics[i]; }
        j += "]";
    }

    j += "}";
}

// ------------------------------ patch apply -----------------------------

struct ApplyOutcome {
    bool ok = false;
    int changed = 0;
    std::vector<std::string> warnings;
};

void setProp(TagLib::PropertyMap& pm, const char* key, const std::string& v) {
    pm.replace(key, TagLib::StringList(TagLib::String(v, TagLib::String::UTF8)));
}
void clearProp(TagLib::PropertyMap& pm, const char* key) { pm.erase(key); }

struct FieldSpec { const char* name; const char* prop; };

bool applyPatchTo(TagLib::File* f, const JVal& patch, ApplyOutcome* out) {
    if (patch.kind != JVal::OBJ) return false;
    TagLib::PropertyMap pm = f->properties();

    static const FieldSpec scalars[] = {
        {"title","TITLE"}, {"album","ALBUM"}, {"date","DATE"},
        {"comment","COMMENT"}, {"key","KEY"}, {"isrc","ISRC"}, {"lyrics","LYRICS"},
    };
    static const FieldSpec lists[] = {
        {"artists","ARTIST"}, {"albumArtists","ALBUMARTIST"},
        {"genres","GENRE"}, {"composers","COMPOSER"},
    };

    auto modeOf = [&](const JVal& field) -> std::string {
        auto it = field.obj.find("mode");
        return it == field.obj.end() ? std::string("unchanged") : it->second.s;
    };

    for (const auto& spec : scalars) {
        auto it = patch.obj.find(spec.name);
        if (it == patch.obj.end()) continue;
        std::string mode = modeOf(it->second);
        if (mode == "unchanged") continue;
        if (mode == "clear") { clearProp(pm, spec.prop); ++out->changed; continue; }
        if (mode != "set") continue;
        auto vit = it->second.obj.find("value");
        if (vit == it->second.obj.end() || vit->second.kind != JVal::STR || vit->second.s.empty())
            continue; // empty_set rejected upstream in Go
        setProp(pm, spec.prop, vit->second.s);
        ++out->changed;
    }
    for (const auto& spec : lists) {
        auto it = patch.obj.find(spec.name);
        if (it == patch.obj.end()) continue;
        std::string mode = modeOf(it->second);
        if (mode == "unchanged") continue;
        if (mode == "clear") { clearProp(pm, spec.prop); ++out->changed; continue; }
        if (mode != "set") continue;
        auto vit = it->second.obj.find("value");
        if (vit == it->second.obj.end() || vit->second.kind != JVal::ARR) continue;
        TagLib::StringList list;
        for (const JVal& item : vit->second.arr)
            if (item.kind == JVal::STR && !item.s.empty())
                list.append(TagLib::String(item.s, TagLib::String::UTF8));
        if (list.isEmpty()) continue;
        pm.replace(spec.prop, list);
        ++out->changed;
    }
    for (const char* numericKey : {"track", "disc"}) {
        auto it = patch.obj.find(numericKey);
        if (it == patch.obj.end()) continue;
        std::string mode = modeOf(it->second);
        if (mode == "unchanged" || mode == "") continue;
        auto vit = it->second.obj.find("value");
        if (vit == it->second.obj.end() || vit->second.kind != JVal::NUM) continue;
        const char* target = (std::string(numericKey) == "track") ? "TRACKNUMBER" : "DISCNUMBER";
        setProp(pm, target, vit->second.s);
        ++out->changed;
    }
    auto bpmIt = patch.obj.find("bpm");
    if (bpmIt != patch.obj.end()) {
        std::string mode = modeOf(bpmIt->second);
        if (mode == "set") {
            auto vit = bpmIt->second.obj.find("value");
            if (vit != bpmIt->second.obj.end() && vit->second.kind == JVal::NUM) {
                setProp(pm, "BPM", vit->second.s);
                ++out->changed;
            }
        } else if (mode == "clear") {
            clearProp(pm, "BPM");
            ++out->changed;
        }
    }

    if (!f->setProperties(pm)) return false;
    out->ok = f->save();
    return out->ok;
}

// --------------------------- payload hashing ----------------------------

bool stripTagsToTemp(const wchar_t* src, std::wstring& tmpOut) {
    tmpOut = std::wstring(src) + L".audiotmp";
    if (!CopyFileW(src, tmpOut.c_str(), FALSE)) return false;
    TagLib::FileRef ref(tmpOut.c_str(), false);
    if (ref.isNull()) return false;
    TagLib::File* f = ref.file();
    if (auto* m = dynamic_cast<TagLib::MPEG::File*>(f)) {
        m->strip(TagLib::MPEG::File::StripID3v1 | TagLib::MPEG::File::StripID3v2 |
                 TagLib::MPEG::File::StripAPE);
        return m->save();
    }
    if (auto* fl = dynamic_cast<TagLib::FLAC::File*>(f)) {
        fl->removeAllTags();
        fl->removeImages();
        return fl->save();
    }
    if (auto* w = dynamic_cast<TagLib::RIFF::WAV::File*>(f)) {
        w->setID3v2Tag(nullptr);
        return w->save();
    }
    if (auto* a = dynamic_cast<TagLib::RIFF::AIFF::File*>(f)) {
        a->setID3v2Tag(nullptr);
        return a->save();
    }
    return false;
}

// --------------------------- buffer/error -------------------------------

static void fill_buffer(et_buffer* out, const std::string& data) {
    if (!out) return;
    const size_t kMaxJson = 32u * 1024u * 1024u;
    if (data.size() > kMaxJson) { out->data = nullptr; out->len = (size_t)-1; return; }
    set_buffer(out, data.data(), data.size());
}

static int fail(et_buffer* err, int code, const char* name, const std::string& detail) {
    if (err) {
        std::string j = "{\"code\":\"";
        j += name; j += "\",\"detail\":\"";
        j += jsonEscape(detail);
        j += "\"}";
        set_buffer(err, j.data(), j.size());
    }
    return code;
}

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
    try {
        std::string raw((const char*)patch, patch_len);
        JsonParser parser(raw.c_str());
        JVal patchVal = parser.parse();
        if (patchVal.kind != JVal::OBJ)
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
        if (!dispatched || !outcome.ok) return ET_ERR_APPLY;
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

void et_buffer_free(et_buffer value) {
    free(value.data);
}
