import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../state/app_state.dart';

class PricingScreen extends StatefulWidget {
  const PricingScreen({super.key});

  @override
  State<PricingScreen> createState() => _PricingScreenState();
}

class _PricingScreenState extends State<PricingScreen> {
  bool _loading = false;
  String? _error;

  Future<void> _upgrade() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final state = context.read<AppState>();
      final res = await state.api.checkoutStripe();
      final urlStr = res['url'] as String?;
      if (urlStr != null) {
        final url = Uri.parse(urlStr);
        if (await canLaunchUrl(url)) {
          await launchUrl(url, mode: LaunchMode.externalApplication);
        } else {
          throw Exception('Could not launch Stripe Checkout URL.');
        }
      } else {
        throw Exception('Server failed to return Checkout URL.');
      }
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final state = context.watch<AppState>();
    final isPro = state.user?['pro_access'] == true;

    return Scaffold(
      backgroundColor: const Color(0xFF0D1117),
      appBar: AppBar(
        title: const Text('Pricing'),
        backgroundColor: const Color(0xFF161B22),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text(
              'Pricing',
              style: TextStyle(
                color: Color(0xFF58A6FF),
                fontWeight: FontWeight.w600,
                letterSpacing: 1.2,
                fontSize: 14,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            const Text(
              'Simple, honest pricing',
              style: TextStyle(
                color: Colors.white,
                fontWeight: FontWeight.bold,
                fontSize: 28,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 12),
            const Text(
              'Start free with 10 notes. Upgrade when Noteika becomes part of your daily workflow.',
              style: TextStyle(
                color: Color(0xFF8B949E),
                fontSize: 14,
                height: 1.45,
              ),
              textAlign: TextAlign.center,
            ),
            if (_error != null) ...[
              const SizedBox(height: 16),
              Text(
                _error!,
                style: const TextStyle(color: Colors.redAccent, fontSize: 14),
                textAlign: TextAlign.center,
              ),
            ],
            const SizedBox(height: 32),
            _buildPlanCard(
              context: context,
              name: 'Free',
              price: '\$0',
              period: 'forever',
              desc: 'Try Noteika — encrypted notes with semantic search.',
              features: [
                '10 encrypted captures',
                'Semantic + exact search',
                'Project folders',
                'Duplicate warnings',
                'JSON export',
              ],
              ctaText: isPro ? 'Downgrade (via Web)' : 'Current Plan',
              isHighlighted: false,
              isCurrent: !isPro,
              onPressed: null,
            ),
            const SizedBox(height: 24),
            _buildPlanCard(
              context: context,
              name: 'Pro',
              price: '\$8',
              period: '/ month',
              desc: 'Unlimited saves for daily use across projects.',
              features: [
                'Unlimited encrypted captures',
                'Semantic + exact search',
                'Everything in Free',
                'Priority support',
                'Cancel anytime',
              ],
              ctaText: isPro ? 'Current Plan' : 'Upgrade to Pro',
              isHighlighted: true,
              isCurrent: isPro,
              onPressed: isPro || _loading ? null : _upgrade,
            ),
            const SizedBox(height: 32),
          ],
        ),
      ),
    );
  }

  Widget _buildPlanCard({
    required BuildContext context,
    required String name,
    required String price,
    required String period,
    required String desc,
    required List<String> features,
    required String ctaText,
    required bool isHighlighted,
    required bool isCurrent,
    required VoidCallback? onPressed,
  }) {
    return Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: const Color(0xFF161B22),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: isHighlighted ? const Color(0xFF58A6FF) : const Color(0xFF30363D),
          width: isHighlighted ? 2.0 : 1.0,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                name,
                style: const TextStyle(
                  color: Colors.white,
                  fontWeight: FontWeight.bold,
                  fontSize: 22,
                ),
              ),
              if (isHighlighted)
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: const Color(0x1F58A6FF),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: const Color(0x6658A6FF)),
                  ),
                  child: const Text(
                    'Unlimited saves',
                    style: TextStyle(
                      color: Color(0xFF58A6FF),
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            crossAxisAlignment: CrossAxisAlignment.baseline,
            textBaseline: TextBaseline.alphabetic,
            children: [
              Text(
                price,
                style: const TextStyle(
                  color: Colors.white,
                  fontWeight: FontWeight.bold,
                  fontSize: 36,
                ),
              ),
              const SizedBox(width: 4),
              Text(
                period,
                style: const TextStyle(
                  color: Color(0xFF8B949E),
                  fontSize: 14,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            desc,
            style: const TextStyle(
              color: Color(0xFF8B949E),
              fontSize: 14,
              height: 1.4,
            ),
          ),
          const SizedBox(height: 24),
          const Divider(color: Color(0xFF30363D)),
          const SizedBox(height: 16),
          ...features.map((feat) => Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: Row(
                  children: [
                    const Icon(Icons.check, color: Color(0xFF3FB950), size: 18),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Text(
                        feat,
                        style: const TextStyle(color: Color(0xFFC9D1D9), fontSize: 14),
                      ),
                    ),
                  ],
                ),
              )),
          const SizedBox(height: 16),
          if (isCurrent)
            OutlinedButton(
              onPressed: null,
              style: OutlinedButton.styleFrom(
                side: const BorderSide(color: Color(0xFF30363D)),
                disabledForegroundColor: const Color(0xFF8B949E),
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
              ),
              child: Text(ctaText),
            )
          else if (onPressed == null)
            ElevatedButton(
              onPressed: null,
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFF21262D),
                disabledForegroundColor: const Color(0xFF8B949E),
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
              ),
              child: Text(ctaText),
            )
          else
            FilledButton(
              onPressed: onPressed,
              style: FilledButton.styleFrom(
                backgroundColor: isHighlighted ? const Color(0xFF1F6FEB) : const Color(0xFF21262D),
                foregroundColor: Colors.white,
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
              ),
              child: _loading
                  ? const SizedBox(
                      height: 18,
                      width: 18,
                      child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                    )
                  : Text(ctaText),
            ),
        ],
      ),
    );
  }
}
