#!/usr/bin/python -i
"""
This does manage to store data on the GPU.
"""

import cPickle
import gzip
import numpy
import os
import theano
import theano.sandbox.cuda as cuda
import theano.tensor as T
import urllib
 
print "%.1fMB free GPU memory" % (cuda.mem_info()[0] / (2.0 ** 20))


# Contains pickled Theano data for digit recognition
MNIST = "http://www.iro.umontreal.ca/~lisa/deep/data/mnist/mnist.pkl.gz"

"""
Gives us a local data path for the data available at the particular URL source.
"""
def data_path(source):
  _, fname = os.path.split(source)
  answer = os.path.abspath(os.path.expanduser("~/data/" + fname))
  if not os.path.isfile(answer):
    # We need to download it
    print "Downloading data from " + source
    urllib.urlretrieve(source, answer)
  return answer

"""
Turns array data into a Theano shared variable so that Theano can
control its storage location and put it on the GPU.
Don't mutate the return value because its dependency guarantees are
unclear.
"""
def make_shared(data):
  assert data.dtype == theano.config.floatX
  return theano.shared(numpy.asarray(data), borrow=True)

"""
Turns int array data into a variable achieved by a cast from an
underlying float array so that the underlying data can be stored on
the GPU.
Don't mutate the return value.
"""
def make_int_shared(data):
  assert str(data.dtype).startswith("int")
  float_data = numpy.asarray(data, dtype=theano.config.floatX)
  shared = theano.shared(float_data, borrow=True)
  return T.cast(shared, "int32")

"""
Returns the MNIST image processing data in normal arrays.
"""
def unshared_mnist():
  f = gzip.open(data_path(MNIST), "rb")
  answer = cPickle.load(f)
  f.close()
  return answer
  
"""
Returns the MNIST image processing data in shared memory.
"""
def mnist():
  ((train_input, train_output),
   (valid_input, valid_output),
   (test_input, test_output)) = unshared_mnist()

  # print "copying training data to GPU"
  s_train_input = make_shared(train_input)
  s_train_output = make_int_shared(train_output)
  # print "copying validation data to GPU"
  s_valid_input = make_shared(valid_input)
  s_valid_output = make_int_shared(valid_output)
  # print "copying testing data to GPU"
  s_test_input = make_shared(test_input)
  s_test_output = make_int_shared(test_output)
  return ((s_train_input, s_train_output),
          (s_valid_input, s_valid_output),
          (s_test_input, s_test_output))


if __name__ == "__main__":
  # See if we have enough memory to load MNIST
  mnist()
