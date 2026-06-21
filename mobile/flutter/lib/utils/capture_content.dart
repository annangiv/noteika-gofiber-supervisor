String generateAutoTitle(String body) {
  final cleaned = body.trim();
  if (cleaned.isEmpty) return 'Untitled Capture';

  final lines = cleaned.split('\n');
  String firstLine = '';
  for (final l in lines) {
    final t = l.trim();
    if (t.isNotEmpty && !t.startsWith('```')) {
      firstLine = t;
      break;
    }
  }

  if (firstLine.isEmpty) {
    for (final l in lines) {
      final t = l.trim();
      if (t.isNotEmpty) {
        firstLine = t.replaceAll('`', '');
        break;
      }
    }
  }

  if (firstLine.isEmpty) firstLine = 'Untitled Capture';

  if (firstLine.length > 80) {
    final sub = firstLine.substring(0, 80);
    final lastSpace = sub.lastIndexOf(' ');
    if (lastSpace > 60) return '${firstLine.substring(0, lastSpace)}...';
    return '$sub...';
  }
  return firstLine;
}

String classifyContentType(String body) {
  final trimmed = body.trim();
  if (trimmed.isEmpty) return 'note';

  if ((trimmed.startsWith('http://') || trimmed.startsWith('https://')) &&
      !trimmed.contains('\n') &&
      !trimmed.contains(' ')) {
    return 'link';
  }
  if (trimmed.contains('```')) return 'code';

  final lower = trimmed.toLowerCase();
  if ((lower.contains('q:') && lower.contains('a:')) ||
      (lower.contains('user:') && lower.contains('assistant:')) ||
      (lower.contains('user:') && lower.contains('ai:')) ||
      (lower.contains('human:') && lower.contains('assistant:'))) {
    return 'qa';
  }
  return 'note';
}

List<String> parseHashtags(String body) {
  final re = RegExp(r'#([a-zA-Z][a-zA-Z0-9_-]{1,31})');
  final seen = <String>{};
  final tags = <String>[];
  for (final m in re.allMatches(body)) {
    final n = m.group(1)!.toLowerCase();
    if (seen.add(n)) tags.add(n);
  }
  tags.sort();
  return tags;
}

List<String> mergeTags(List<List<String>> lists) {
  final seen = <String>{};
  final out = <String>[];
  for (final list in lists) {
    for (final t in list) {
      final n = t.trim().toLowerCase().replaceFirst(RegExp(r'^#'), '');
      if (n.length >= 2 && seen.add(n)) out.add(n);
    }
  }
  out.sort();
  return out.length > 12 ? out.sublist(0, 12) : out;
}

String formatRelativeTime(int epochSeconds) {
  final date = DateTime.fromMillisecondsSinceEpoch(epochSeconds * 1000);
  final diff = DateTime.now().difference(date);
  if (diff.inMinutes < 1) return 'just now';
  if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
  if (diff.inHours < 24) return '${diff.inHours}h ago';
  if (diff.inDays == 1) return 'yesterday';
  if (diff.inDays < 7) return '${diff.inDays}d ago';
  return '${date.month}/${date.day}/${date.year}';
}
