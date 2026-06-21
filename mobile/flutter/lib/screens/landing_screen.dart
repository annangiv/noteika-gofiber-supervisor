import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../state/app_state.dart';

class LandingScreen extends StatefulWidget {
  const LandingScreen({super.key});

  @override
  State<LandingScreen> createState() => _LandingScreenState();
}

class _LandingScreenState extends State<LandingScreen> {
  final PageController _pageController = PageController();
  int _currentPage = 0;

  final List<Map<String, dynamic>> _features = [
    {
      'title': 'Prevent Duplicates',
      'tagline': 'Notes that resurface before you duplicate yourself.',
      'desc': 'Noteika checks text similarity in real-time as you draft snippets, warning you instantly if similar thoughts already exist in your docket.',
      'icon': Icons.copy_all_rounded,
      'color': const Color(0xFFD4A32D), // Amber
    },
    {
      'title': 'Semantic Search',
      'tagline': 'Search by meaning, not just exact keywords.',
      'desc': 'A quantized BGE-small neural network runs on-device. Search queries are mapped to vectors to find semantically related content instantly.',
      'icon': Icons.psychology_rounded,
      'color': const Color(0xFF58A6FF), // Blue
    },
    {
      'title': 'E2E Zero-Knowledge',
      'tagline': 'Your knowledge remains exclusively yours.',
      'desc': 'Plaintext notes and vector embeddings are encrypted on-device with AES-GCM-256 before syncing. Noteika servers never see your notes.',
      'icon': Icons.security_rounded,
      'color': const Color(0xFF3FB950), // Green
    },
  ];

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final state = context.watch<AppState>();

    return Scaffold(
      backgroundColor: const Color(0xFF0D1117),
      body: SafeArea(
        child: Column(
          children: [
            const SizedBox(height: 32),
            // Brand Header
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: const Color(0xFF161B22),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: const Color(0xFF30363D)),
                  ),
                  child: ShaderMask(
                    shaderCallback: (bounds) => const LinearGradient(
                      colors: [Color(0xFF58A6FF), Color(0xFFBC8CFF)],
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                    ).createShader(bounds),
                    child: const Icon(
                      Icons.blur_on_rounded,
                      size: 28,
                      color: Colors.white,
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                const Text(
                  'Noteika',
                  style: TextStyle(
                    color: Colors.white,
                    fontSize: 26,
                    fontWeight: FontWeight.bold,
                    letterSpacing: 0.8,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Text(
              'Save once, find by meaning.',
              style: TextStyle(
                color: const Color(0xFF8B949E),
                fontSize: 14,
                letterSpacing: 0.5,
              ),
            ),
            
            Expanded(
              child: PageView.builder(
                controller: _pageController,
                onPageChanged: (page) {
                  setState(() {
                    _currentPage = page;
                  });
                },
                itemCount: _features.length,
                itemBuilder: (context, index) {
                  final feature = _features[index];
                  return SingleChildScrollView(
                    child: Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          // Animated Glow Icon
                          TweenAnimationBuilder<double>(
                            tween: Tween(begin: 0.0, end: 1.0),
                            duration: const Duration(milliseconds: 600),
                            builder: (context, value, child) {
                              return Transform.scale(
                                scale: 0.8 + (value * 0.2),
                                child: Opacity(
                                  opacity: value,
                                  child: child,
                                ),
                              );
                            },
                            child: Container(
                              height: 100,
                              width: 100,
                              decoration: BoxDecoration(
                                color: feature['color'].withOpacity(0.08),
                                shape: BoxShape.circle,
                                border: Border.all(
                                  color: feature['color'].withOpacity(0.3),
                                  width: 2,
                                ),
                                boxShadow: [
                                  BoxShadow(
                                    color: feature['color'].withOpacity(0.1),
                                    blurRadius: 20,
                                    spreadRadius: 2,
                                  ),
                                ],
                              ),
                              child: Icon(
                                feature['icon'],
                                size: 48,
                                color: feature['color'],
                              ),
                            ),
                          ),
                          const SizedBox(height: 24),
                          // Feature Title
                          Text(
                            feature['title'],
                            style: const TextStyle(
                              color: Colors.white,
                              fontSize: 22,
                              fontWeight: FontWeight.bold,
                            ),
                            textAlign: TextAlign.center,
                          ),
                          const SizedBox(height: 12),
                          // Feature Tagline
                          Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 16),
                            child: Text(
                              feature['tagline'],
                              style: TextStyle(
                                color: feature['color'],
                                fontSize: 14,
                                fontWeight: FontWeight.w600,
                                height: 1.3,
                              ),
                              textAlign: TextAlign.center,
                            ),
                          ),
                          const SizedBox(height: 10),
                          // Feature Description
                          Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 12),
                            child: Text(
                              feature['desc'],
                              style: const TextStyle(
                                color: Color(0xFF8B949E),
                                fontSize: 13,
                                height: 1.4,
                              ),
                              textAlign: TextAlign.center,
                            ),
                          ),
                        ],
                      ),
                    ),
                  );
                },
              ),
            ),
            
            // Page Indicator Dots
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: List.generate(
                _features.length,
                (index) => AnimatedContainer(
                  duration: const Duration(milliseconds: 250),
                  margin: const EdgeInsets.symmetric(horizontal: 4),
                  height: 8,
                  width: _currentPage == index ? 24 : 8,
                  decoration: BoxDecoration(
                    color: _currentPage == index
                        ? const Color(0xFF58A6FF)
                        : const Color(0xFF30363D),
                    borderRadius: BorderRadius.circular(4),
                  ),
                ),
              ),
            ),
            const SizedBox(height: 32),

            // Navigation Button / CTA
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: SizedBox(
                width: double.infinity,
                height: 52,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(10),
                    gradient: const LinearGradient(
                      colors: [Color(0xFF1F6FEB), Color(0xFF58A6FF)],
                    ),
                  ),
                  child: ElevatedButton(
                    onPressed: () => state.markLandingSeen(),
                    style: ElevatedButton.styleFrom(
                      backgroundColor: Colors.transparent,
                      shadowColor: Colors.transparent,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(10),
                      ),
                    ),
                    child: const Text(
                      'Get Started',
                      style: TextStyle(
                        color: Colors.white,
                        fontSize: 16,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                ),
              ),
            ),
            const SizedBox(height: 16),
            
            // Secondary Skip to Login Action
            TextButton(
              onPressed: () => state.markLandingSeen(),
              child: const Text(
                'Already have an account? Sign in',
                style: TextStyle(
                  color: Color(0xFF58A6FF),
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }
}
