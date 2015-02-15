#!/usr/bin/env luajit

require "nn"
require "torch"

mnistTrain = torch.load("mnist.t7/train_32x32.t7", "ascii")
mnistTest = torch.load("mnist.t7/test_32x32.t7", "ascii")

-- A Dataset can be either training or testing.
Dataset = {}
function Dataset:new(data, labels)
  local dataset = {
    original=data,
    labels=labels
  }
  setmetatable(dataset, {__index = Dataset})
  return dataset
end

-- Creates a Dataset from an mnist-format input that has "data" and
-- "labels".
function Dataset.makeTraining(abnormal)
  local dataset = Dataset:new(abnormal.data, abnormal.labels)
  dataset.normalized = torch.FloatTensor(dataset.original:size())
  dataset.normalized:copy(dataset.original)
  dataset.mean = dataset.normalized:mean()
  dataset.std = dataset.normalized:std()
  dataset.normalized:add(-dataset.mean)
  dataset.normalized:div(dataset.std)
  return dataset
end

-- Makes a new dataset using the same transformation by which dataset
-- was originally created.
-- abnormal should have "data" and "labels".
function Dataset:makeTest(abnormal)
  local test = Dataset:new(abnormal.data, abnormal.labels)
  test.normalized = torch.FloatTensor(test.original:size())
  test.normalized:copy(test.original)
  test.normalized:add(-self.mean)
  test.normalized:div(self.std)
  return test
end

-- A Net is a neural net with helper functions
Net = {}
function Net:new(trainingDataset)
  local net = {
    train=trainingDataset,
  }
  setmetatable(net, {__index = Net})

  -- The model to train
  local ninputs = net.train.normalized:stride(1)
  net.model = nn.Sequential()
  net.model:add(nn.Reshape(ninputs))
  net.model:add(nn.Linear(ninputs, 10))
  net.model:add(nn.LogSoftMax())

  net.criterion = nn.ClassNLLCriterion()

  return net
end

-- Trains on a single input-output pair.
-- input should be a tensor with the input data
-- TODO: is either 1x32x32 or 32x32 ok?
-- TODO: does this work?
-- label should just be a number with the digit+1 (stupid 1-indexing)
function Net:trainOnce(input, label)
  local output = torch.Tensor(1)
  self.criterion:forward(self.model:forward(input), output)
  self.model:zeroGradParameters()
  self.model:backward(
    input, self.criterion:backward(self.model.output, output))
  self.model:updateParameters(0.01)
end

function Net:trainIndex(i)
  self:trainOnce(self.train.normalized[i], self.train.labels[i])
end

function Net:trainAll()
  for i = 1,self.train:size(1) do
    self:trainOnce(i)
  end
end

function Net:classify(input)
  -- TODO
end

train = Dataset.makeTraining(mnistTrain)
test = train:makeTest(mnistTest)
net = Net:new(train)

-- Ghetto testing
assert(string.format("%.4f", test.normalized[3][1][4][2]) == "-0.3635")
