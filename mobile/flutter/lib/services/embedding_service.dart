import 'dart:io';
import 'dart:math' as math;
import 'dart:typed_data';
import 'package:dart_wordpiece/dart_wordpiece.dart';
import 'package:dio/dio.dart';
import 'package:flutter_onnxruntime/flutter_onnxruntime.dart';
import 'package:path_provider/path_provider.dart';

class EmbeddingService {
  static const String _modelUrl =
      'https://huggingface.co/Xenova/bge-small-en-v1.5/resolve/main/onnx/model_quantized.onnx';
  static const String _vocabUrl =
      'https://huggingface.co/Xenova/bge-small-en-v1.5/resolve/main/vocab.txt';

  OrtSession? _session;
  WordPieceTokenizer? _tokenizer;
  bool loaded = false;

  Future<bool> checkModelExists() async {
    final dir = await getApplicationDocumentsDirectory();
    final modelFile = File('${dir.path}/bge_model/model.onnx');
    final vocabFile = File('${dir.path}/bge_model/vocab.txt');
    return await modelFile.exists() && await vocabFile.exists();
  }

  Future<void> downloadAndLoadModel(Function(double) onProgress) async {
    if (loaded) return;

    final dir = await getApplicationDocumentsDirectory();
    final modelDir = Directory('${dir.path}/bge_model');
    if (!await modelDir.exists()) {
      await modelDir.create(recursive: true);
    }

    final modelFile = File('${modelDir.path}/model.onnx');
    final vocabFile = File('${modelDir.path}/vocab.txt');

    final dio = Dio();

    if (!await vocabFile.exists()) {
      await dio.download(_vocabUrl, vocabFile.path);
    }

    if (!await modelFile.exists()) {
      await dio.download(
        _modelUrl,
        modelFile.path,
        onReceiveProgress: (received, total) {
          if (total > 0) {
            onProgress(received / total);
          }
        },
      );
    } else {
      onProgress(1.0);
    }

    await loadModel(modelFile.path, vocabFile.path);
  }

  Future<void> loadModel(String modelPath, String vocabPath) async {
    final vocabContent = await File(vocabPath).readAsString();
    final vocab = VocabLoader.fromString(vocabContent);
    _tokenizer = WordPieceTokenizer(vocab: vocab);

    final ort = OnnxRuntime();
    _session = await ort.createSession(modelPath);
    loaded = true;
  }

  Future<List<double>> embedText(String text, {required bool isQuery}) async {
    if (!loaded || _session == null || _tokenizer == null) {
      throw StateError('Embedding model is not loaded');
    }

    final prefix = isQuery ? 'query: ' : 'passage: ';
    final textToEmbed = '$prefix${text.trim().toLowerCase()}';

    final output = _tokenizer!.encode(textToEmbed);
    var inputIds = output.inputIds;
    var attentionMask = output.attentionMask;

    // Truncate to BGE max sequence length of 512 tokens
    if (inputIds.length > 512) {
      inputIds = inputIds.sublist(0, 512);
      attentionMask = attentionMask.sublist(0, 512);
    }

    final tokenTypeIds = List<int>.filled(inputIds.length, 0);

    final inputTensor = await OrtValue.fromList(
      Int64List.fromList(inputIds),
      [1, inputIds.length],
    );
    final maskTensor = await OrtValue.fromList(
      Int64List.fromList(attentionMask),
      [1, attentionMask.length],
    );
    
    // Check if the loaded ONNX model expects token_type_ids
    final hasTypeIds = _session!.inputNames.contains('token_type_ids');
    OrtValue? typeTensor;
    if (hasTypeIds) {
      typeTensor = await OrtValue.fromList(
        Int64List.fromList(tokenTypeIds),
        [1, tokenTypeIds.length],
      );
    }

    final inputs = {
      'input_ids': inputTensor,
      'attention_mask': maskTensor,
      if (hasTypeIds && typeTensor != null) 'token_type_ids': typeTensor,
    };

    final outputs = await _session!.run(inputs);
    final lastHiddenState = outputs['last_hidden_state'] ?? outputs.values.first;
    final flatData = await lastHiddenState.asFlattenedList();

    // Clean up input tensors immediately
    inputTensor.dispose();
    maskTensor.dispose();
    typeTensor?.dispose();

    // Clean up output tensors
    for (final v in outputs.values) {
      v.dispose();
    }

    // Mean Pooling using attention mask
    final dim = 384;
    final seqLen = attentionMask.length;
    final pooled = List<double>.filled(dim, 0.0);
    double maskSum = 0.0;

    for (var i = 0; i < seqLen; i++) {
      final maskVal = attentionMask[i].toDouble();
      maskSum += maskVal;
      final offset = i * dim;
      for (var d = 0; d < dim; d++) {
        pooled[d] += flatData[offset + d] * maskVal;
      }
    }

    if (maskSum > 0.0) {
      for (var d = 0; d < dim; d++) {
        pooled[d] /= maskSum;
      }
    }

    // L2 Normalization
    double norm = 0.0;
    for (var d = 0; d < dim; d++) {
      norm += pooled[d] * pooled[d];
    }
    norm = math.sqrt(norm);

    if (norm > 0.0) {
      for (var d = 0; d < dim; d++) {
        pooled[d] /= norm;
      }
    }

    return pooled;
  }

  Future<void> dispose() async {
    await _session?.close();
    _session = null;
    _tokenizer = null;
    loaded = false;
  }
}
