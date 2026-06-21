import 'package:flutter/material.dart';

class AboutScreen extends StatelessWidget {
  const AboutScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('About Noteika'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back_ios_new, size: 18),
          onPressed: () => Navigator.pop(context),
        ),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text(
              'ABOUT',
              style: TextStyle(
                color: Color(0xFFA78BFA),
                fontWeight: FontWeight.w600,
                letterSpacing: 1.5,
                fontSize: 12,
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
                letterSpacing: -0.5,
              ),
            ),
            const SizedBox(height: 16),
            const Text(
              'Most note apps are great at saving and terrible at bringing notes back when they matter. Noteika does one job: save once, find later — even if you forgot the name.',
              style: TextStyle(
                color: Color(0xFF9CA3AF),
                fontSize: 15,
                height: 1.5,
              ),
            ),
            const SizedBox(height: 32),
            _buildSectionCard(
              title: 'The Problem',
              description:
                  'Prompts live in AI chats. Links live in browser tabs. Decisions live in scratchpads. A week later you know you figured it out — but not where you put it.',
            ),
            const SizedBox(height: 16),
            _buildSectionCard(
              title: 'Our Approach',
              description:
                  'One capture box. Silent type detection for display. Semantic search instead of keyword guessing. Proactive resurfacing so you see duplicates before you redo the work.',
            ),
            const SizedBox(height: 16),
            _buildSectionCard(
              title: "What We're Not",
              description:
                  'Not a complex graph view. Not team sharing. Not in-app AI chat. Noteika is the place you save from elsewhere and find when it counts.',
            ),
            const SizedBox(height: 32),
            const Divider(color: Color(0xFF1F2228), height: 1),
            const SizedBox(height: 24),
            const Text(
              'How it works',
              style: TextStyle(
                color: Colors.white,
                fontWeight: FontWeight.bold,
                fontSize: 20,
                letterSpacing: -0.3,
              ),
            ),
            const SizedBox(height: 16),
            _buildStepRow('1', 'Paste anything — prompt, link, Q&A, code snippet.'),
            const SizedBox(height: 16),
            _buildStepRow('2', 'Tag a project optionally (Keller, client work, Inbox…).'),
            const SizedBox(height: 16),
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
        color: const Color(0xFF13151A),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: const Color(0xFF1F2228)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            title,
            style: const TextStyle(
              color: Colors.white,
              fontWeight: FontWeight.bold,
              fontSize: 16,
              letterSpacing: -0.3,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            description,
            style: const TextStyle(
              color: Color(0xFF9CA3AF),
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
          decoration: BoxDecoration(
            color: const Color(0x1F8B5CF6),
            shape: BoxShape.circle,
            border: Border.all(color: const Color(0xFF8B5CF6), width: 1.5),
          ),
          child: Text(
            num,
            style: const TextStyle(
              color: Color(0xFFA78BFA),
              fontWeight: FontWeight.bold,
              fontSize: 13,
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Text(
            text,
            style: const TextStyle(
              color: Color(0xFF9CA3AF),
              fontSize: 14,
              height: 1.45,
            ),
          ),
        ),
      ],
    );
  }
}

