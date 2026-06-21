import 'package:flutter/material.dart';

class AboutScreen extends StatelessWidget {
  const AboutScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF0D1117),
      appBar: AppBar(
        title: const Text('About Noteika'),
        backgroundColor: const Color(0xFF161B22),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text(
              'About',
              style: TextStyle(
                color: Color(0xFF58A6FF),
                fontWeight: FontWeight.w600,
                letterSpacing: 1.2,
                fontSize: 14,
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              'Built for builders, not archivists',
              style: TextStyle(
                color: Colors.white,
                fontWeight: FontWeight.bold,
                fontSize: 28,
                height: 1.2,
              ),
            ),
            const SizedBox(height: 16),
            const Text(
              'Most note apps are great at saving and terrible at bringing notes back when they matter. Noteika does one job: save once, find later — even if you forgot the name.',
              style: TextStyle(
                color: Color(0xFF8B949E),
                fontSize: 16,
                height: 1.5,
              ),
            ),
            const SizedBox(height: 32),
            _buildSectionCard(
              title: 'The problem',
              description:
                  'Prompts live in AI chats. Links live in browser tabs. Decisions live in scratchpads. A week later you know you figured it out — but not where you put it.',
            ),
            const SizedBox(height: 16),
            _buildSectionCard(
              title: 'Our approach',
              description:
                  'One capture box. Silent type detection for display. Semantic search instead of keyword guessing. Proactive resurfacing so you see duplicates before you redo the work.',
            ),
            const SizedBox(height: 16),
            _buildSectionCard(
              title: 'What we\'re not',
              description:
                  'Not a graph view. Not team sharing. Not in-app AI chat. Noteika is the place you save from elsewhere and find when it counts.',
            ),
            const SizedBox(height: 32),
            const Divider(color: Color(0xFF30363D)),
            const SizedBox(height: 24),
            const Text(
              'How it works',
              style: TextStyle(
                color: Colors.white,
                fontWeight: FontWeight.bold,
                fontSize: 20,
              ),
            ),
            const SizedBox(height: 16),
            _buildStepRow('1', 'Paste anything — prompt, link, Q&A, code snippet.'),
            const SizedBox(height: 12),
            _buildStepRow('2', 'Tag a project optionally (Keller, client work, Inbox…).'),
            const SizedBox(height: 12),
            _buildStepRow('3', 'Find by meaning when you need it — or get nudged if it already exists.'),
            const SizedBox(height: 32),
          ],
        ),
      ),
    );
  }

  Widget _buildSectionCard({required String title, required String description}) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: const Color(0xFF161B22),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: const Color(0xFF30363D)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            title,
            style: const TextStyle(
              color: Colors.white,
              fontWeight: FontWeight.bold,
              fontSize: 18,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            description,
            style: const TextStyle(
              color: Color(0xFF8B949E),
              fontSize: 14,
              height: 1.5,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildStepRow(String num, String text) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          width: 28,
          height: 28,
          alignment: Alignment.center,
          decoration: const BoxDecoration(
            color: Color(0xFF1F6FEB),
            shape: BoxShape.circle,
          ),
          child: Text(
            num,
            style: const TextStyle(
              color: Colors.white,
              fontWeight: FontWeight.bold,
              fontSize: 14,
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Text(
            text,
            style: const TextStyle(
              color: Color(0xFF8B949E),
              fontSize: 14,
              height: 1.4,
            ),
          ),
        ),
      ],
    );
  }
}
