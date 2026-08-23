//
// Length-aware UI-string lookup for the Engine DJ translation proxy.
//
// Pure, portable, header-only: no Windows types, no heap allocation,
// noexcept throughout. Nothing here assumes NUL termination -- every
// function takes an explicit byte length and never reads past it.
//
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <optional>
#include <string_view>

namespace etproxy {

inline constexpr std::size_t kHashTableSize = 8192;

// Longest key in the translation table. Any input longer than this cannot
// match, so lookup() rejects it without scanning the extra bytes.
inline constexpr std::size_t kMaxKeyLength = 128;

// Documented maximum scan length for lookup_bounded() when the caller passes
// a negative size (meaning "NUL-terminated, length unknown"). Callers who
// pass a negative size assert that at least one readable byte lies within
// this distance; the wrapper still never relies on finding a terminator.
inline constexpr std::size_t kMaxBoundedScan = 4096;

// At most this many bytes are rendered into log output per string.
inline constexpr std::size_t kMaxLogDisplay = 120;

struct TranslationEntry {
  std::string_view en;
  std::string_view cn;
};

// English source fragment -> simplified-Chinese UI label. Keys must stay
// unique and no longer than kMaxKeyLength bytes.
inline constexpr TranslationEntry kTranslations[] = {
    {"Preferences", "偏好设置"}, {"About", "关于"}, {"Exit", "退出"},
    {"Collection", "收藏集"}, {"Playlists", "播放列表"}, {"History", "历史记录"},
    {"Prepare", "准备列表"}, {"Files", "文件"},
    {"Browse", "浏览"}, {"Search", "搜索"}, {"Library", "曲库"},
    {"Settings", "设置"}, {"Tools", "工具"},
    {"OK", "确定"}, {"Cancel", "取消"}, {"Save", "保存"}, {"Delete", "删除"},
    {"Rename", "重命名"}, {"Close", "关闭"}, {"Back", "返回"},
    {"Next", "下一步"}, {"Finish", "完成"}, {"Apply", "应用"}, {"Reset", "重置"},
    {"Play", "播放"}, {"Pause", "暂停"}, {"Stop", "停止"}, {"Cue", "提示点"},
    {"Loop", "循环"}, {"Sync", "同步"}, {"Quantize", "量化"}, {"Slip", "滑移"},
    {"Master", "主控"}, {"Track", "曲目"}, {"Artist", "艺术家"},
    {"Album", "专辑"}, {"Genre", "流派"}, {"Duration", "时长"}, {"Key", "调性"},
    {"BPM", "BPM"}, {"Loading...", "加载中..."},
    {"Yes", "是"}, {"No", "否"}, {"Import", "导入"}, {"Export", "导出"},
    {"Refresh", "刷新"}, {"All Tracks", "全部曲目"},
    {"Recently Added", "最近添加"}, {"Most Played", "最多播放"},
    {"New Playlist", "新建播放列表"}, {"Create Playlist", "创建播放列表"},
    {"Analyze", "分析"}, {"Re-analyze", "重新分析"},
    {"General", "通用"}, {"Audio", "音频"},
    {"Advanced", "高级"}, {"Volume", "音量"}, {"EQ", "均衡器"},
    {"Elapsed", "已播放"}, {"Remaining", "剩余"}, {"Total", "总计"},
    {"Sort by", "排序方式"}, {"View", "视图"}, {"Grid", "网格"}, {"List", "列表"},
    {"Add to Playlist", "添加到播放列表"}, {"Edit", "编辑"},
    {"Copy", "复制"}, {"Paste", "粘贴"}, {"Select All", "全选"},
    {"Deck 1", "碟机 1"}, {"Deck 2", "碟机 2"},
    {"Deck 3", "碟机 3"}, {"Deck 4", "碟机 4"},
    {"Sampler", "采样器"}, {"Record", "录音"}, {"Mic", "麦克风"},
    {"Error", "错误"}, {"Warning", "警告"},
    {"Information", "信息"}, {"Success", "完成"},
    {"Sync Manager", "同步管理器"},
    {"Export to USB", "导出到 USB"}, {"Import from USB", "从 USB 导入"},
};

inline std::uint32_t fnv1a(const char* data, std::size_t length) noexcept {
  std::uint32_t h = 2166136261u;
  for (std::size_t i = 0; i < length; ++i) {
    h ^= static_cast<unsigned char>(data[i]);
    h *= 16777619u;
  }
  return h;
}

struct HashTable {
  const TranslationEntry* slots[kHashTableSize] = {};
  std::size_t count = 0;
};

inline const HashTable& hash_table() noexcept {
  static const HashTable table = [] {
    HashTable t;
    for (const auto& entry : kTranslations) {
      if (entry.en.empty()) continue;
      std::size_t h = fnv1a(entry.en.data(), entry.en.size()) % kHashTableSize;
      while (t.slots[h] != nullptr) h = (h + 1) % kHashTableSize;
      t.slots[h] = &entry;
      ++t.count;
    }
    return t;
  }();
  return table;
}

// Looks up a string by its exact byte range [data, data + length).
// Returns std::nullopt when there is no match; never dereferences past
// `length` bytes and never assumes a terminator.
inline std::optional<std::string_view> lookup(const char* data,
                                              std::size_t length) noexcept {
  if (data == nullptr || length == 0 || length > kMaxKeyLength) {
    return std::nullopt;
  }
  const HashTable& t = hash_table();
  std::size_t h = fnv1a(data, length) % kHashTableSize;
  for (std::size_t probes = 0; probes < kHashTableSize && t.slots[h] != nullptr;
       ++probes) {
    const std::string_view key = t.slots[h]->en;
    if (key.size() == length &&
        std::memcmp(key.data(), data, length) == 0) {
      return t.slots[h]->cn;
    }
    h = (h + 1) % kHashTableSize;
  }
  return std::nullopt;
}

// Size-aware front door for the hooks. Non-negative sizes are exact byte
// lengths (no NUL assumption). Negative sizes mean "NUL-terminated, length
// unknown" and are accepted ONLY here: they scan at most kMaxBoundedScan
// bytes looking for the terminator, and never rely on finding one.
inline std::optional<std::string_view> lookup_bounded(
    const char* data, std::int64_t size) noexcept {
  if (data == nullptr || size == 0) return std::nullopt;
  if (size < 0) {
    std::size_t n = 0;
    while (n < kMaxBoundedScan && data[n] != '\0') ++n;
    return lookup(data, n);
  }
  return lookup(data, static_cast<std::size_t>(size));
}

// Heuristic: could this byte range be user-visible UI text? Fully
// length-bounded; used only to decide what is worth logging.
inline bool looks_like_ui(const char* data, std::size_t length) noexcept {
  if (length < 2 || length > 200) return false;
  for (std::size_t i = 0; i < length; ++i) {
    const unsigned char c = static_cast<unsigned char>(data[i]);
    if ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80 ||
        c == ' ' || c == '.' || c == '-' || c == '\'' || c == '/' ||
        c == '&' || c == '(' || c == ')' || c == ':') {
      continue;
    }
    return false;
  }
  return true;
}

// Escapes an untrusted byte range into `out` as a quoted, NUL-terminated
// string safe to hand to printf-family "%s". Reads at most
// min(size, kMaxLogDisplay) bytes (negative sizes follow the
// lookup_bounded convention), stops at an embedded NUL, escapes control
// bytes as \xNN and truncates with "..." when needed. Returns the number
// of characters written, excluding the terminating NUL.
inline std::size_t quote_into(char* out, std::size_t capacity,
                              const char* data, std::int64_t size) noexcept {
  if (out == nullptr || capacity == 0) return 0;
  std::size_t pos = 0;
  bool truncated = false;

  const auto room = [&]() -> bool {
    if (pos + 1 >= capacity) {
      truncated = true;
      return false;
    }
    return true;
  };
  const auto put = [&](char c) {
    if (room()) out[pos++] = c;
  };
  const auto hex = [&](unsigned char c) {
    static const char kDigits[] = "0123456789abcdef";
    put('\\');
    put('x');
    put(kDigits[(c >> 4) & 0xF]);
    put(kDigits[c & 0xF]);
  };

  put('"');
  if (data != nullptr) {
    const std::size_t available =
        size < 0 ? kMaxBoundedScan : static_cast<std::size_t>(size);
    const std::size_t limit =
        available < kMaxLogDisplay ? available : kMaxLogDisplay;
    for (std::size_t i = 0; i < limit; ++i) {
      const unsigned char c = static_cast<unsigned char>(data[i]);
      if (c == '\0') break;  // stop silently at an embedded terminator
      switch (c) {
        case '"': put('\\'); put('"'); break;
        case '\\': put('\\'); put('\\'); break;
        case '\n': put('\\'); put('n'); break;
        case '\r': put('\\'); put('r'); break;
        case '\t': put('\\'); put('t'); break;
        default:
          if (c < 0x20 || c == 0x7F) {
            hex(c);
          } else {
            put(static_cast<char>(c));
          }
          break;
      }
      if (truncated) break;
    }
  }
  if (truncated) {
    put('.'); put('.'); put('.');
  }
  put('"');
  out[pos] = '\0';
  return pos;
}

}  // namespace etproxy
