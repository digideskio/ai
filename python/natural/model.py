#!/usr/bin/env python3
import random
import tensorflow as tf
import numpy as np

from tensorflow.models.rnn import rnn_cell
from tensorflow.models.rnn import seq2seq

MIN_NUMBER = 1
MAX_NUMBER = 127
def number():
    return random.randrange(MIN_NUMBER, MAX_NUMBER + 1)

# Format is like
# 10*10
# 100
# space-padded on the right to get consistent lengths.
SOURCE_VOCAB, TARGET_VOCAB = '10* ', '10 '
SOURCE_LEN = 1 + 2 * len('{0:b}'.format(MAX_NUMBER))
TARGET_LEN = len('{0:b}'.format(MAX_NUMBER * MAX_NUMBER))

def source_pad(s):
  while len(s) <= SOURCE_LEN:
    s += ' '
  return s

def target_pad(s):
  while len(s) <= TARGET_LEN:
    s += ' '
  return s

  
# Generates one example (source, target) pair
def generate():
  a, b = number(), number()
  c = a * b
  source = source_pad('{0:b}*{0:b}'.format(a, b))
  target = target_pad('{0:b}'.format(c))
  assert all(ch in SOURCE_VOCAB for ch in source)
  assert all(ch in TARGET_VOCAB for ch in target)
  assert len(source) == SOURCE_LEN
  assert len(target) == TARGET_LEN
  return source, target
  
    
class Model(object):

  '''
  If learning=False, that means we are not training, and we don't need to
  learn things on the fly.
  '''
  def __init__(self, learning=True):

    # Set up hyperparameters
    self.num_layers = 2
    self.layer_size = 128
    self.batch_size = 50

    # Set up the core RNN cells of the tensor network
    single_cell = rnn_cell.BasicLSTMCell(self.layer_size)
    self.cell = rnn_cell.MultiRNNCell([single_cell] * self.num_layers)

    # Set up placeholders for the source and target embeddings
    self.encoder_inputs = [tf.placeholder(tf.int32,
                                          shape=[self.batch_size],
                                          name='encoder{0}'.format(i))
                           for i in range(SOURCE_LEN)]
    
    self.decoder_inputs = [tf.placeholder(tf.int32,
                                          shape=[self.batch_size],
                                          name='decoder{0}'.format(i))
                           for i in range(TARGET_LEN)]

    # Weights for the decoding
    self.decoder_weights = [tf.ones([self.batch_size], tf.float32)
                            for i in range(TARGET_LEN)]
    
    # Construct the seq2seq part of the model
    # For what exactly outputs and states are, see
    # https://github.com/tensorflow/tensorflow/blob/master/tensorflow/python/ops/seq2seq.py
    self.outputs, self.states = seq2seq.embedding_rnn_seq2seq(
      self.encoder_inputs,
      self.decoder_inputs,
      self.cell,
      len(SOURCE_VOCAB),
      len(TARGET_VOCAB))

    self.losses = seq2seq.sequence_loss_by_example(
      self.outputs,
      self.decoder_inputs,
      self.decoder_weights)
      
      

    
